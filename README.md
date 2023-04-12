# gsmtp
GSMTP => Go Simple Mail Transfer Protocol

It transparently act as a simple local mail server (not authentication required), and does the tunneling to 
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
  -c string
        Config file path (default "config.yml")

```

Sample `config.yml`: Refer to `config.example.yml`

Example:
```yml
gmail_username: my-gmail@gmail.com
gmail_password: bbbb
port: 25
tls_port: 455
bind_address: 127.0.0.2
cert_file: cert1
key_file: key1
```

Parameters can be override by env variables. See comments below.

gmail_username: The gmail sender address. Can be override by `GMAIL_USERNAME`

gmail_password: The gmail app password. Can be override by `GMAIL_PASSWORD`

port: The plaintext port to use. Can be override by `PORT`. Default is `25`

tls_port: The secure port to use. Can be override by `TLS_PORT`. Default is `465`

bind_address: The bind address of the service. Can be override by `BIND`. Default is `127.0.0.1`

cert_file: TLS Cert file path. Can be override by `CERT_FILE`. Default is ""

key_file: TLS Key file path. Can be override by `KEY_FILE`. Default is ""

Secure port is not enabled unless both `cert_file` and `key_file` are provided.

## Running
```
$ ./gsmtp -c config.yml
```

We provided a `localhost.pem` and `localhost.key`, for you to test.

# Behavior
It will inject login credential to google automatically after first succesful `HELO` or `EHLO` call.

Configure your mail client as no auth. 

Port 25 by default is the plain port

Port 465 is the TLS port. 

