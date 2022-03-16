# gsmtp
Local SMTP that sends mail through gmail without authentication

# Building
```
$ go build
```

# Running
## Prepare GMAIL account
You need to get a gmail application password first.
See https://support.google.com/accounts/answer/185833?hl=en

## Prepare TLS

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
# Behavior
It will inject login credential to google automatically after first succesful `HELO` or `EHLO` call.

Configure your mail client as no auth. 

Port 25 by default is the plain port
Port 465 is the TLS port (to use this, you need to have a tls cert and key, then pass the file name with the
command line switch)