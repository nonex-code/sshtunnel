package core

import (
	"golang.org/x/crypto/ssh"
	"io"
	"log"
	"net"
)

// SSHTunnelConfig 用于配置SSH隧道的相关信息
type SSHTunnelConfig struct {
	SSHClientConfig      ssh.ClientConfig // SSH客户端配置
	SSHServerAddress     string           // SSH服务器地址
	RemoteListenAddress  string           // 远程服务器的监听地址
	LocalServiceAddress  string           // 本地服务地址
	RemoteForwardAddress string           // 远程端口转发地址
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

// ForwardLocalPort 在远程服务器上监听指定的端口并处理连接(将本地端口转发到远程)
func (config *SSHTunnelConfig) ForwardLocalPort() {
	sshClient, err := config.Connect()
	if err != nil {
		log.Fatalf("无法连接到SSH服务器: %s", err)
	}
	defer sshClient.Close()

	// 在远程服务器上监听指定的端口
	listener, err := sshClient.Listen("tcp", config.RemoteListenAddress)
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
		go config.handleLocalForwarding(remoteConn)
	}
}

// handleLocalForwarding 处理到远程连接的请求并转发到本地服务
func (config *SSHTunnelConfig) handleLocalForwarding(remoteConn net.Conn) {
	defer remoteConn.Close()

	// 建立到本地服务的连接
	localConn, err := net.Dial("tcp", config.LocalServiceAddress)
	if err != nil {
		log.Printf("无法连接到本地服务: %s", err)
		return
	}
	defer localConn.Close()

	// 转发数据
	go io.Copy(localConn, remoteConn)
	io.Copy(remoteConn, localConn)
}

// ForwardRemotePort 将远程端口转发到本地
func (config *SSHTunnelConfig) ForwardRemotePort() {
	if config.RemoteForwardAddress == "" {
		log.Println("远程转发地址未设置，跳过远程端口转发。")
		return
	}

	// 建立 SSH 连接
	sshClient, err := config.Connect()
	if err != nil {
		log.Fatalf("无法连接到SSH服务器: %s", err)
	}
	defer sshClient.Close()

	log.Printf("将远程端口 %s 转发到本地 %s", config.RemoteForwardAddress, config.LocalServiceAddress)

	// 在本地监听
	listener, err := net.Listen("tcp", config.LocalServiceAddress)
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
		go config.handleRemoteForwarding(sshClient, localConn)
	}
}

// handleRemoteForwarding 处理远程转发连接
func (config *SSHTunnelConfig) handleRemoteForwarding(sshClient *ssh.Client, localConn net.Conn) {
	defer localConn.Close()

	remoteConn, err := sshClient.Dial("tcp", config.RemoteForwardAddress)
	if err != nil {
		log.Printf("无法连接到远程转发地址: %s", err)
		return
	}
	defer remoteConn.Close()

	// 转发数据
	go io.Copy(remoteConn, localConn)
	io.Copy(localConn, remoteConn)
}
