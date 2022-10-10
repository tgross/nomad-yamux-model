package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hashicorp/yamux"
)

func main() {
	flag.Parse()
	args := flag.Args()

	if len(args) == 0 {
		log.Fatal("first argument must be server or client")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGHUP)
		<-sigCh
		cancel()
		log.Print("shutting down in 1sec...")
		time.Sleep(1 * time.Second)
		log.Print("bye!")
		os.Exit(0)
		return
	}()

	switch args[0] {
	case "server":
		addr := "127.0.0.1:7078"
		if len(args) > 1 {
			addr = args[1]
		}
		server(ctx, addr)
	case "client":
		if len(args) < 2 {
			log.Fatal("second argument must be an address")
		}
		client(ctx, args[1:])
	default:
		log.Fatal("first argument must be server or client")
	}
}

func client(ctx context.Context, addrs []string) {
	session, err := createSession(addrs)
	if err != nil {
		log.Fatal(err)
	}
	t := time.NewTimer(time.Second)
	for {
		select {
		case <-ctx.Done():
			session.Close()
			return
		case <-t.C:
			err := clientSend(session)
			if err != nil {
				log.Printf("send error: %v, reconnecting to next server", err)
				session.Close()
				badAddr := addrs[0]
				addrs = addrs[1:]
				addrs = append(addrs, badAddr)
				session, err = createSession(addrs)
				if err != nil {
					log.Fatal(err)
				}
			}
			t.Reset(time.Second)
		}
	}
}

func createSession(addrs []string) (*yamux.Session, error) {
	conn, err := net.Dial("tcp", addrs[0])
	if err != nil {
		return nil, err
	}
	cfg := yamux.DefaultConfig()
	cfg.KeepAliveInterval = time.Second * 30

	return yamux.Client(conn, nil)
}

func clientSend(session *yamux.Session) error {
	stream, err := session.Open()
	if err != nil {
		return fmt.Errorf("failed to open session: %v", err)

	}
	defer stream.Close()
	wrote, err := stream.Write([]byte("ping"))
	if err != nil {
		return fmt.Errorf("failed to write: %v", err)
	}
	if wrote < 4 {
		return fmt.Errorf("short write %d bytes", wrote)
	}
	log.Print("wrote ping...")
	buf := make([]byte, 4)
	_, err = stream.Read(buf)
	if err != nil {
		return fmt.Errorf("read error: %v\n", err)
	}
	log.Print("got pong!")

	return nil
}

func server(_ context.Context, addr string) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		panic(err)
	}
	listener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		panic(err)
	}

	log.Printf("listening for incoming connections on %s", addr)

	// NOTE: this is wrong but is also how Nomad does it currently! We
	// should fix that!
	ctx := context.Background()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			conn, err := listener.Accept()
			if err != nil {
				panic(err)
			}
			go handleConn(ctx, conn)
		}
	}

}

func handleConn(ctx context.Context, conn net.Conn) {
	session, err := yamux.Server(conn, nil)
	defer session.Close()
	if err != nil {
		panic(err)
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
			stream, err := session.Accept()
			if err != nil {
				log.Printf("accept error: %v", err)
			}

			go handleStream(stream)
		}
	}
}

func handleStream(stream net.Conn) {
	buf := make([]byte, 4)
	defer stream.Close()
	_, err := stream.Read(buf)
	if err != nil {
		log.Printf("read error: %v", err)
		return
	}
	log.Print("pong")
	_, err = stream.Write([]byte("pong"))
	if err != nil {
		log.Printf("write error: %v", err)
		return
	}
}