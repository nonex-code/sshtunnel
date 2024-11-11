package core

import (
	"golang.org/x/crypto/ssh"
	"io"
	"log"
	"net"
)

type TunnelConf struct {
	Config           ssh.ClientConfig
	RemoteServerAddr string
	RemoteLicAddr    string
	LocalServerAddr  string
}

func (s *TunnelConf) ConnectAndForward() {
	log.Printf("远程监听地址：%s , 本地服务地址：%s", s.RemoteLicAddr, s.LocalServerAddr)
	// 建立 SSH 连接
	conn, err := ssh.Dial("tcp", s.RemoteServerAddr, &s.Config)
	if err != nil {
		log.Fatalf("Failed to dial: %s", err)
	}
	defer conn.Close()
	log.Printf("Successfully connected to %s", s.RemoteLicAddr)
	s.listenAndForward(conn)
}

func (s *TunnelConf) listenAndForward(conn *ssh.Client) {
	// 在远程服务器上监听指定的端口
	listener, err := conn.Listen("tcp", s.RemoteLicAddr)
	if err != nil {
		log.Fatalf("Failed to listen on remote: %s", err)
	}
	defer listener.Close()
	log.Println("Waiting for connections...")
	for {
		remoteConn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept remote connection: %s", err)
			continue
		}
		go s.handleConnection(remoteConn)
	}
}

func (s *TunnelConf) handleConnection(remoteConn net.Conn) {
	defer remoteConn.Close()
	// 建立到本地服务的连接
	localConn, err := net.Dial("tcp", s.LocalServerAddr)
	if err != nil {
		log.Printf("Failed to connect to local service: %s", err)
		return
	}
	defer localConn.Close()
	// 转发数据
	go io.Copy(localConn, remoteConn)
	io.Copy(remoteConn, localConn)
}
