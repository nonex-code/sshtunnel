package core

import (
	"golang.org/x/crypto/ssh"
	"io"
	"log"
	"net"
)

// SSHTunnelConfig 用于配置SSH隧道的相关信息
type SSHTunnelConfig struct {
	SSHConfig           ssh.ClientConfig // SSH客户端配置
	SSHServerAddr       string           // SSH服务器地址
	RemoteListeningAddr string           // 远程服务器的监听地址
	LocalServiceAddr    string           // 本地服务地址
}

// Connect 建立SSH连接并开始监听和转发
func (s *SSHTunnelConfig) Connect() {
	// 建立 SSH 连接
	conn, err := ssh.Dial("tcp", s.SSHServerAddr, &s.SSHConfig)
	if err != nil {
		log.Fatalf("Failed to dial: %s", err)
	}
	defer conn.Close()
	log.Printf("Successfully connected to %s", s.RemoteListeningAddr)
	s.listenAndForward(conn)
}

// listenAndForward 在远程服务器上监听指定的端口并处理连接
func (s *SSHTunnelConfig) listenAndForward(conn *ssh.Client) {
	// 在远程服务器上监听指定的端口
	listener, err := conn.Listen("tcp", s.RemoteListeningAddr)
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

// handleConnection 处理到远程连接的请求并转发到本地服务
func (s *SSHTunnelConfig) handleConnection(remoteConn net.Conn) {
	defer remoteConn.Close()
	// 建立到本地服务的连接
	localConn, err := net.Dial("tcp", s.LocalServiceAddr)
	if err != nil {
		log.Printf("Failed to connect to local service: %s", err)
		return
	}
	defer localConn.Close()
	// 转发数据
	go io.Copy(localConn, remoteConn)
	io.Copy(remoteConn, localConn)
}
