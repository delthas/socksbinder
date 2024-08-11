package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
)

var bind string
var listen string

func handle(c net.Conn) error {
	defer c.Close()

	b := bufio.NewReader(c)
	v, err := b.ReadByte()
	if err != nil {
		return err
	}
	if v != 0x05 {
		return fmt.Errorf("unknown socks version: %x", v)
	}
	n, err := b.ReadByte()
	if err != nil {
		return err
	}
	var hasNoAuth bool
	for i := 0; i < int(n); i++ {
		m, err := b.ReadByte()
		if err != nil {
			return err
		}
		if m == 0x00 {
			hasNoAuth = true
		}
	}
	if !hasNoAuth {
		return fmt.Errorf("client does not support using no authentication")
	}
	if _, err := c.Write([]byte{0x05, 0x00}); err != nil {
		return err
	}
	v, err = b.ReadByte()
	if err != nil {
		return err
	}
	if v != 0x05 {
		return fmt.Errorf("unknown socks version: %x", v)
	}
	m, err := b.ReadByte()
	if err != nil {
		return err
	}
	if m != 0x01 {
		return fmt.Errorf("unknown socks command, want CONNECT: %x", m)
	}
	if _, err := b.Discard(1); err != nil {
		return err
	}
	t, err := b.ReadByte()
	if err != nil {
		return err
	}
	var ip net.IP
	switch t {
	case 0x01: // IPv4
		ip = make([]byte, 4)
		if _, err := io.ReadFull(b, ip); err != nil {
			return err
		}
	case 0x04: // IPv6
		ip = make([]byte, 16)
		if _, err := io.ReadFull(b, ip); err != nil {
			return err
		}
	case 0x03: // hostname
		n, err := b.ReadByte()
		if err != nil {
			return err
		}
		host := make([]byte, n)
		if _, err := io.ReadFull(b, host); err != nil {
			return err
		}
		hip, err := net.ResolveIPAddr("ip", string(host))
		if err != nil {
			return fmt.Errorf("resolving %q: %v", string(host), err)
		}
		ip = hip.IP
	default:
		return fmt.Errorf("unknown address type: %x", t)
	}
	pb := make([]byte, 2)
	if _, err := io.ReadFull(b, pb); err != nil {
		return err
	}
	port := binary.BigEndian.Uint16(pb)

	var laddr *net.TCPAddr
	if bind != "" {
		laddr = &net.TCPAddr{
			IP: net.ParseIP(bind),
		}
	}
	raddr := &net.TCPAddr{
		IP:   ip,
		Port: int(port),
	}
	o, err := net.DialTCP("tcp", laddr, raddr)
	if err != nil {
		return fmt.Errorf("dialing %q: %v", raddr, err)
	}
	defer func() {
		o.Close()
	}()

	var r bytes.Buffer
	r.WriteByte(0x05)
	r.WriteByte(0x00)
	r.WriteByte(0x00)

	laddr = o.LocalAddr().(*net.TCPAddr)
	if ip4 := laddr.IP.To4(); ip4 != nil { // IPv4
		r.WriteByte(0x01)
		r.Write(ip4)
	} else { // IPv6
		r.WriteByte(0x04)
		r.Write(ip)
	}

	lpb := make([]byte, 2)
	binary.BigEndian.PutUint16(lpb, uint16(laddr.Port))
	r.Write(lpb)

	if _, err := c.Write(r.Bytes()); err != nil {
		return err
	}

	log.Printf("proxying %q -> %q", c.RemoteAddr(), o.RemoteAddr())

	mr := io.MultiReader(io.LimitReader(b, int64(b.Buffered())), c)

	done := make(chan error, 2)
	go func() {
		_, err := io.Copy(o, mr)
		done <- err
	}()
	go func() {
		_, err := io.Copy(c, o)
		done <- err
	}()
	<-done
	return nil
}

func main() {
	flag.StringVar(&bind, "bind", "", "which local address to bind outgoing connections to")
	flag.StringVar(&listen, "listen", ":1080", "which address:port to listen on")
	flag.Parse()

	l, err := net.Listen("tcp", listen)
	if err != nil {
		log.Fatalf("listening on %q: %v", listen, err)
		return
	}

	log.Printf("listening on %q", listen)

	for {
		c, err := l.Accept()
		if err != nil {
			log.Printf("accepting: %v", err)
			continue
		}
		go func() {
			if err := handle(c); err != nil {
				log.Print(err)
			}
		}()
	}
}
