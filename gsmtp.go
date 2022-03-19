package main

import (
	"bufio"
	"crypto/tls"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

var sequence int64 = 0
var port_number = flag.Int("port", 25, "The smtp server port")
var secure_port_number = flag.Int("secure-port", 465, "The smtp secure port with tls")
var bind_address = flag.String("bind", "", "The bind address. Defaults to all interface")
var cert = flag.String("tls-cert", "", "The TLS cert")
var key = flag.String("tls-key", "", "The TLS Key")
var verbose = flag.Bool("verbose", false, "Show debug or not")
var sigs = make(chan os.Signal, 1)
var stop = false
var gmail_user = os.Getenv("GMAIL_USERNAME")
var gmail_password = os.Getenv("GMAIL_PASSWORD")
var credential = genCredential(gmail_user, gmail_password)

const CRLF = "\r\n"

var active_client_count int64 = 0

const RHOST = "smtp.gmail.com:465"

func printFlags() {
	fmt.Printf("Config: port                  => %d\n", *port_number)
	fmt.Printf("Config: secure-port           => %d\n", *secure_port_number)
	fmt.Printf("Config: bind                  => %s\n", *bind_address)
	fmt.Printf("Config: tls-cert              => %s\n", *cert)
	fmt.Printf("Config: tls-key               => %s\n", *key)
	fmt.Printf("Config: verbose               => %t\n", *verbose)
}

func main() {
	flag.Parse()
	printFlags()

	if gmail_user == "" || gmail_password == "" {
		log.Printf("NEED GMAIL_USERNAME and GMAIL_PASSWORD. you can do:" +
			"\n  export GMAIL_USERNAME=xxxx\n  export GMAIL_PASSWORD=yyy")
		log.Fatal("Please specify a GMAIL_USERNAME and GMAIL_PASSWORD using environment variable")
	}
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		// This goroutine executes a blocking receive for
		// signals. When it gets one it'll print it out
		// and then notify the program that it can finish.
		sig := <-sigs
		log.Println(">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>ALERT<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<")
		log.Println(sig)
		log.Println("Waiting for graceful shutdown - Press CTRL-C or kill again to force quit")
		stop = true
		signal.Stop(sigs)
	}()
	var wg sync.WaitGroup
	if *port_number != -1 {
		listen_address := fmt.Sprintf("%s:%d", *bind_address, *port_number)

		normal, err := net.Listen("tcp", listen_address)
		if err != nil {
			log.Fatalf("Failed to start normal listener on %s. Error: %s", listen_address, err)
		}
		wg.Add(1)
		go handle(normal, &wg)
	}

	if *secure_port_number != -1 && *cert != "" && *key != "" {
		x509_cert, err := tls.LoadX509KeyPair(*cert, *key)

		if err != nil {
			log.Fatalf("Failed to load cert/key pair. cert: %s key: %s, error: %s", *cert, *key, err)
		}
		config := tls.Config{Certificates: []tls.Certificate{x509_cert}}

		listen_address := fmt.Sprintf("%s:%d", *bind_address, *secure_port_number)

		secure, err := tls.Listen("tcp", listen_address, &config)
		if err != nil {
			log.Fatalf("Failed to start secure listener on %s. Error: %s", listen_address, err)
		}
		wg.Add(1)
		go handle(secure, &wg)
	}

	wg.Wait()
	waitCounter := 0
	for {
		if active_client_count == 0 {
			break
		}
		if waitCounter%50 == 0 {
			log.Printf("%d client(s) active\n", active_client_count)
		}
		time.Sleep(100 * time.Millisecond)
		waitCounter++
	}
	log.Printf("Bye.")
}
func listenWithChannel(listener net.Listener, channel chan net.Conn) {
	for !stop {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		channel <- conn
	}
	close(channel)
}
func handle(listener net.Listener, wg *sync.WaitGroup) {
	log.Printf("Started listener on address %s", listener.Addr())
	defer wg.Done()
	socks := make(chan net.Conn, 50000)
	go listenWithChannel(listener, socks)

	newwg := sync.WaitGroup{}
	for !stop {
		select {
		case conn, ok := <-socks:
			if !ok {
				break
			}
			newwg.Add(1)
			atomic.AddInt64(&active_client_count, 1)
			go handleConn(conn, &newwg)
		case <-time.After(500 * time.Millisecond):
		}
	}
	listener.Close()
	log.Printf("Stopped listener on address %s", listener.Addr())
}

const WELCOME = "220 simple.smtp.server welcomes you" + CRLF

func handleConn(conn net.Conn, wg *sync.WaitGroup) {
	defer func() {
		atomic.AddInt64(&active_client_count, -1)
		conn.Close()
		log.Printf("Done handling connection from %+v", conn.RemoteAddr())
	}()
	log.Printf("Handling connection from %+v", conn.RemoteAddr())

	defer wg.Done()
	buff := make([]byte, 4096)
	tlsCfg := tls.Config{ServerName: "smtp.gmail.com"}
	rconn, err := tls.Dial("tcp", RHOST, &tlsCfg)
	if err != nil {
		log.Printf("Connect to %s failed: %s", RHOST, err)
		conn.Close()
		return
	}
	defer rconn.Close()

	nread, err := rconn.Read(buff)
	if err != nil {
		log.Printf("GMAIL read error: %s", err)
		return
	}
	nwritten, err := conn.Write(buff[:nread])
	if err != nil || nwritten != nread {
		log.Printf("Write error:%s, written %d(expect %d)", err, nwritten, nread)
		return
	}
	for {
		nread, err = conn.Read(buff)
		if err != nil {
			log.Printf("Read from client err: %s", err)
			return
		}
		var line string = string(buff[:nread])
		upperLine := strings.ToUpper(line)
		isHELO := false
		HELOPositive := false
		if strings.HasPrefix(upperLine, "HELO") || strings.HasPrefix(upperLine, "EHLO") {
			isHELO = true
		}
		nwritten, err = rconn.Write(buff[:nread])
		if err != nil || nwritten != nread {
			log.Printf("Write to gmail failed: %s - written %d (expect %d)", err, nwritten, nread)
			return
		}
		nread, err = rconn.Read(buff)
		resp := string(buff[:nread])
		if strings.HasPrefix(resp, "250") {
			HELOPositive = true
		}
		if err != nil {
			log.Printf("Read from gmail failed: %s", err)
			return
		}
		nwritten, err = conn.Write(buff[:nread])
		if err != nil || nwritten != nread {
			log.Printf("Write to client failed: %s - written %d (expect %d)", err, nwritten, nread)
			return
		}
		if isHELO && HELOPositive {
			break
		}
	}
	//
	var credential = "AUTH PLAIN " + credential + CRLF
	log.Printf("Injecting login credentials")
	rconn.Write([]byte(credential))
	nread, err = rconn.Read(buff)
	if err != nil {
		log.Printf("Error reading from gmail %s", err)
		return
	}
	loginResult := string(buff[:nread])
	if !strings.HasPrefix(loginResult, "235") {
		log.Printf("Gmail login failed: %s", loginResult)
		return
	}

	nwg := sync.WaitGroup{}
	nwg.Add(2)
	go pipe(conn, rconn, &nwg)
	go pipe(rconn, conn, &nwg)
	nwg.Wait()
}

// Setup a pipe, read from a reader and write to writer immediately.
// After it is done, reduce wait group by 1
func pipe(r io.ReadCloser, w io.WriteCloser, wg *sync.WaitGroup) {
	defer wg.Done()
	buff := make([]byte, 1024)
	for {
		nread, err := r.Read(buff)
		if err != nil {
			r.Close()
			w.Close()
			return
		}
		nwritten, err := w.Write(buff[:nread])
		if err != nil || nwritten != nread {
			r.Close()
			w.Close()
			return
		}
	}
}

func readLineFrom(reader *bufio.Reader, buffer []byte) (string, error) {
	readCount := 0
	remaining := len(buffer)
	run := true
	for run {
		bytes, isPrefix, err := reader.ReadLine()
		if err != nil {
			return "", err
		}
		toCopy := min(len(bytes), remaining)
		copyBytes(buffer, readCount, bytes, 0, toCopy)
		readCount += toCopy
		remaining -= toCopy
		if !isPrefix {
			run = false
		}
	}
	return string(buffer[:readCount]), nil
}

// Copy bytes from src to dest, start at srcOffset and destOffset, for count bytes
func copyBytes(dest []byte, destOffset int, src []byte, srcOffset int, count int) {
	for i := 0; i < count; i++ {
		dest[destOffset+i] = src[srcOffset+i]
	}
}

// Find the smaller item in the two
func min(a1 int, a2 int) int {
	if a1 < a2 {
		return a1
	}
	return a2
}

// Convert a user + password => base64('\0' + user + '\0' + password)
func genCredential(user string, pass string) string {
	userBytes := []byte(user)
	passBytes := []byte(pass)
	buf := make([]byte, len(userBytes)+len(passBytes)+2)
	buf[0] = 0
	for idx, b := range userBytes {
		buf[idx+1] = b
	}

	lenUserBytes := len(userBytes)
	buf[lenUserBytes+1] = 0
	for idx, b := range passBytes {
		buf[idx+2+lenUserBytes] = b
	}

	return base64.StdEncoding.EncodeToString(buf)
}
