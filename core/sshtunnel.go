package core

import (
	"golang.org/x/crypto/ssh"
	"io"
	"log"
	"net"
)

// SSHTunnelConfig 用于配置SSH隧道的相关信息
type SSHTunnelConfig struct {
	SSHClientConfig  ssh.ClientConfig // SSH客户端配置
	SSHServerAddress string           // SSH服务器地址
}

// Connect 建立SSH连接
func (config *SSHTunnelConfig) Connect() (*ssh.Client, error) {
	// 建立 SSH 连接
	sshClient, err := ssh.Dial("tcp", config.SSHServerAddress, &config.SSHClientConfig)
	if err != nil {
		return nil, err
	}
	log.Printf("成功连接到SSH服务器 %s", config.SSHServerAddress)
	return sshClient, nil
}

// ForwardLocalPortToRemote 在远程服务器上监听指定的端口并处理连接(将本地端口转发到远程)
func (config *SSHTunnelConfig) ForwardLocalPortToRemote(remoteListenAddress, localServiceAddress string) {
	sshClient, err := config.Connect()
	if err != nil {
		log.Fatalf("无法连接到SSH服务器: %s", err)
	}
	defer sshClient.Close()

	// 在远程服务器上监听指定的端口
	listener, err := sshClient.Listen("tcp", remoteListenAddress)
	if err != nil {
		log.Fatalf("无法在远程服务器上监听: %s", err)
	}
	defer listener.Close()

	log.Println("等待传入连接...")
	for {
		remoteConn, err := listener.Accept()
		if err != nil {
			log.Printf("无法接受远程连接: %s", err)
			continue
		}
		go config.handleLocalForwarding(remoteConn, localServiceAddress)
	}
}

// handleLocalForwarding 处理到远程连接的请求并转发到本地服务
func (config *SSHTunnelConfig) handleLocalForwarding(remoteConn net.Conn, localServiceAddress string) {
	defer remoteConn.Close()

	// 建立到本地服务的连接
	localConn, err := net.Dial("tcp", localServiceAddress)
	if err != nil {
		log.Printf("无法连接到本地服务: %s", err)
		return
	}
	defer localConn.Close()

	// 转发数据
	go io.Copy(localConn, remoteConn)
	io.Copy(remoteConn, localConn)
}

// ForwardRemotePortToLocal 将远程端口转发到本地
func (config *SSHTunnelConfig) ForwardRemotePortToLocal(remoteForwardAddress, localListenAddress string) {
	if remoteForwardAddress == "" {
		log.Println("远程转发地址未设置，跳过远程端口转发。")
		return
	}

	// 建立 SSH 连接
	sshClient, err := config.Connect()
	if err != nil {
		log.Fatalf("无法连接到SSH服务器: %s", err)
	}
	defer sshClient.Close()

	log.Printf("将远程端口 %s 转发到本地 %s", remoteForwardAddress, localListenAddress)

	// 在本地监听
	listener, err := net.Listen("tcp", localListenAddress)
	if err != nil {
		log.Fatalf("无法在本地监听: %s", err)
	}
	defer listener.Close()

	for {
		localConn, err := listener.Accept()
		if err != nil {
			log.Printf("无法接受本地连接: %s", err)
			continue
		}
		go config.handleRemoteForwarding(sshClient, localConn, remoteForwardAddress)
	}
}

// handleRemoteForwarding 处理远程转发连接
func (config *SSHTunnelConfig) handleRemoteForwarding(sshClient *ssh.Client, localConn net.Conn, remoteForwardAddress string) {
	defer localConn.Close()

	remoteConn, err := sshClient.Dial("tcp", remoteForwardAddress)
	if err != nil {
		log.Printf("无法连接到远程转发地址: %s", err)
		return
	}
	defer remoteConn.Close()

	// 转发数据
	go io.Copy(remoteConn, localConn)
	io.Copy(localConn, remoteConn)
}
