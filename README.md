# socksbinder

A simple SOCKSv5 proxy enabling you to bind outgoing connections to a specific network interface.

Given some program `foobar` that wants to connect to `example.com:12345`, that you want to bind to `10.2.0.1`:

- Run socksbinder: `socksbinder -bind 10.2.0.1 -listen :8080`
- Run foobar, adding socksbinder as a SOCKSv5 proxy: `ALL_PROXY=socks5://127.0.0.1:8080 foobar`

`foobar` will then connect to `example.com:12345` through your local `socksbinder`, which will bind the connection to `10.2.0.1`.

## Installing

Requires Go.

```shell
go install github.com/delthas/socksbinder@latest
```

## Usage

Example usage, telling `socksbinder` to listen on `8080`, binding outgoing network connections to `10.2.0.1`:

```shell
socksbinder -bind 10.2.0.1 -listen :8080
```

For details, see `socksbinder -help`.

## License

MIT
