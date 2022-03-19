# gsmtp
GSMTP => Go Simple Mail Transfer Protocol

It transparently act as a simple local mail server (not authentication required), and does the tunning to 
GMAIL SMTP and does the authentication for you.

Local SMTP that sends mail through gmail without authentication. Why?

Postfix and Exim configuration are quite complicated, and 90% of the hosts don't need them.
However you may still want to be able to send out emails to external on cronjob results.

This program is for you.

You may want to replace `/usr/sbin/sendmail` because it depends on the postfix unfortunately.

Something like this will work (`mv /usr/sbin/sendmail /usr/sbin/sendmail.old`)

Create /usr/sbin/sendmail with following content

```perl
#!/usr/bin/perl
use IO::Socket::INET;
`date >> /tmp/sendmail.log`;
my $ARGC = 0 + @ARGV;
my $recipient = "";
foreach my $arg(@ARGV) {
        if($arg =~ m/^.*\@.*$/g) {
                $recipient = $arg;
                last;
        }
}
die "no recipient found" if not $recipient;

$| = 1;
my $socket = new IO::Socket::INET (
    PeerHost => 'localhost',
    PeerPort => '25',
    Proto => 'tcp',
);

my $helo = "HELO home.wushilin.net\r\n";
&send($socket, $helo);
&receive($socket);
&send($socket, "MAIL from: <wushilin.sg\@gmail.com>\r\n");
&receive($socket);
&send($socket, "RCPT to: <$recipient>\r\n");
&receive($socket);
&send($socket, "DATA\r\n");
while(<STDIN>) {
        chomp;
        &send($socket, $_ . "\r\n");
}
&send($socket, ".\r\n");
&receive($socket);
&send($socket, "quit\r\n");
&receive($socket);
$socket->close();

sub send($$) {
        my $sock = shift;
        my $data = shift;
        $sock->send($data);
}

sub receive($) {
        my $sock = shift;
        my $response = "";
        $sock->recv($response, 1024);
}
```

# Building
```
$ go build
```

# Running
## Prepare GMAIL account
You need to get a gmail application password first.
See https://support.google.com/accounts/answer/185833?hl=en

## Prepare TLS
Generate your key and certs in openssl, or alternatively, checkout

https://github.com/wushilin/minica

## Getting help
You can run with `-h` switch to see what is supported

```
./gsmtp -h
Usage of ./gsmtp:
  -bind string
        The bind address. Defaults to all interface
  -port int
        The smtp server port (default 25)
  -secure-port int
        The smtp secure port with tls (default 465)
  -tls-cert string
        The TLS cert
  -tls-key string
        The TLS Key
  -verbose
        Show debug or not

```

## Running
```
$ export GMAIL_USER your-gmail@gmail.com
$ export GMAIL_PASSWORD you-gmail-app-password
$ ./gsmtp
```

We provided a `localhost.pem` and `localhost.key`, for you to test. to use it, please run with
```sh
$ ./gsmtp -tls-cert localhost.pem -tls-key localhost.key
```

If you don't want to bind default host or port, use
```sh
$ ./gsmtp -bind 172.20.0.3 -port 10025 -secure-port 10465 -tls-cert localhost.pem -tls-key localhost.key
```
# Behavior
It will inject login credential to google automatically after first succesful `HELO` or `EHLO` call.

Configure your mail client as no auth. 

Port 25 by default is the plain port

Port 465 is the TLS port. To use this, you need to have a tls cert and key, then pass the file name with the
command line switch (see example above)

