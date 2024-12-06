package main

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	"os"
	"runtime"
	"sshtunnel/core"
	"strings"
)

// 定义全局变量
var (
	commit = "unknown" // 默认提交 ID
	branch = "unknown" // 默认分支名称
)

// 预定义常量
const (
	SSHAddressFormatError          = "SSH服务地址格式错误，应为 user@host:port"
	ErrMissingSSHAddress           = "错误：缺少SSH地址。请使用 -s 选项指定 SSH 服务地址，例如: -s user@host:port"
	ErrMissingRemoteAddr           = "错误：缺少远程地址。请使用 -r 选项指定远程服务器监听地址，例如: -r 127.0.0.1:2000"
	ErrMissingLocalAddr            = "错误：缺少本地地址。请使用 -l 选项指定本地服务地址，例如: -l 127.0.0.1:5000"
	ErrInvalidPortForwardDirection = "错误：方向参数无效。请使用 '-R' 将远程端口转发到本地，或 '-L' 将本地端口转发到远程。"
)

// 定义根命令
var rootCmd = &cobra.Command{
	Use:   "sshtunnel",
	Short: "SSH隧道创建工具",
	Long:  `sshtunnel 是一个用于创建SSH隧道的命令行工具，支持使用密钥或密码进行认证`,
}

// 添加版本信息命令
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "显示版本信息",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("%s-%s %s %s/%s ", branch, commit, runtime.Version(), runtime.GOOS, runtime.GOARCH)
	},
}

// 初始化
func init() {
	// 添加标志参数
	rootCmd.Flags().StringP("ssh", "s", "", "SSH服务地址：user@host:port (必需)")
	rootCmd.Flags().StringP("remote", "r", "", "远程服务器监听地址：host:port (必需)")
	rootCmd.Flags().StringP("local", "l", "", "本地服务地址：host:port (必需)")
	rootCmd.Flags().StringP("auth", "a", "", "认证信息，使用方式：key:/path/to/key或直接密码")

	// 添加方向标志
	rootCmd.Flags().BoolP("remote-local", "R", false, "将远程端口转发到本地")
	rootCmd.Flags().BoolP("local-remote", "L", false, "将本地端口转发到远程")
	// 添加版本命令
	rootCmd.AddCommand(versionCmd)

	rootCmd.Run = executePortForwarding
}

// 执行端口转发
func executePortForwarding(cmd *cobra.Command, args []string) {
	isRemoteToLocal, _ := cmd.Flags().GetBool("remote-local")
	isLocalToRemote, _ := cmd.Flags().GetBool("local-remote")

	if !(isRemoteToLocal || isLocalToRemote) {
		cmd.Help() // 如果未指定转发方向，显示帮助信息
		return
	}

	if isRemoteToLocal {
		if err := initiateRemoteToLocalForwarding(); err != nil {
			fmt.Println(err) // 输出错误信息
		}
	}

	if isLocalToRemote {
		if err := initiateLocalToRemoteForwarding(); err != nil {
			fmt.Println(err) // 输出错误信息
		}
	}
}

func initiateRemoteToLocalForwarding() error {
	return initiatePortForwarding("remote")
}

func initiateLocalToRemoteForwarding() error {
	return initiatePortForwarding("local")
}

func initiatePortForwarding(direction string) error {
	sshAddress, _ := rootCmd.Flags().GetString("ssh")
	remoteAddress, _ := rootCmd.Flags().GetString("remote")
	localAddress, _ := rootCmd.Flags().GetString("local")
	authenticationInfo, _ := rootCmd.Flags().GetString("auth")

	if err := verifyInputParams(sshAddress, remoteAddress, localAddress, direction); err != nil {
		return err // 返回验证错误
	}

	username, serverAddress, err := parseSSHAddress(sshAddress) // 解析SSH地址
	if err != nil {
		return err // 返回解析错误
	}

	sshPrivateKey, sshPassword := parseAuthentication(authenticationInfo) // 解析认证信息
	fmt.Printf("使用用户: %s，SSH地址: %s\n", username, serverAddress)

	if direction == "remote" {
		fmt.Printf("开始将远程%s转发到本地%s\n", remoteAddress, localAddress)
		return forwardRemotePortToLocal(username, sshPrivateKey, sshPassword, serverAddress, remoteAddress, localAddress)
	} else {
		fmt.Printf("开始将本地%s转发到远程%s\n", localAddress, remoteAddress)
		return forwardLocalPortToRemote(username, sshPrivateKey, sshPassword, serverAddress, localAddress, remoteAddress)
	}
}

// 处理远程到本地的端口转发
func forwardRemotePortToLocal(username, privateKeyPath, password, serverAddress, remoteAddr, localAddr string) error {
	// 处理SSH隧道的创建逻辑
	sshConfig, err := createSSHClientConfig(username, privateKeyPath, password)
	if err != nil {
		return err
	}

	tunnelConfig := core.SSHTunnelConfig{
		SSHClientConfig:  sshConfig,
		SSHServerAddress: serverAddress,
	}
	tunnelConfig.ForwardRemotePortToLocal(remoteAddr, localAddr) // 将远程端口转发到本地
	return nil                                                   // 确保返回错误值
}

// 处理本地到远程的端口转发
func forwardLocalPortToRemote(username, privateKeyPath, password, serverAddress, localAddr, remoteAddr string) error {
	// 处理SSH隧道的创建逻辑
	sshConfig, err := createSSHClientConfig(username, privateKeyPath, password)
	if err != nil {
		return err
	}

	tunnelConfig := core.SSHTunnelConfig{
		SSHClientConfig:  sshConfig,
		SSHServerAddress: serverAddress,
	}
	tunnelConfig.ForwardLocalPortToRemote(remoteAddr, localAddr) // 将本地端口转发到远程
	return nil                                                   // 确保返回错误值
}

// 创建SSH客户端配置
func createSSHClientConfig(username, privateKeyPath, password string) (ssh.ClientConfig, error) {
	var authMethods []ssh.AuthMethod

	// 处理密钥认证
	if privateKeyPath != "" {
		privateKey, err := os.ReadFile(privateKeyPath)
		if err != nil {
			return ssh.ClientConfig{}, fmt.Errorf("密钥读取失败: %v", err)
		}
		privateKeySigner, err := ssh.ParsePrivateKey(privateKey)
		if err != nil {
			return ssh.ClientConfig{}, fmt.Errorf("密钥解析失败: %v", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(privateKeySigner))
	}

	// 处理密码认证
	if password != "" {
		authMethods = append(authMethods, ssh.Password(password))
	}

	sshConfig := ssh.ClientConfig{
		User:            username,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 注意：生产环境中使用安全的 HostKeyCallback
	}

	return sshConfig, nil
}

// 验证输入参数
func verifyInputParams(sshAddress, remoteAddress, localAddress, direction string) error {
	if sshAddress == "" {
		return errors.New(ErrMissingSSHAddress)
	}
	if remoteAddress == "" {
		return errors.New(ErrMissingRemoteAddr)
	}
	if localAddress == "" {
		return errors.New(ErrMissingLocalAddr)
	}
	if direction != "remote" && direction != "local" {
		return errors.New(ErrInvalidPortForwardDirection)
	}
	return nil
}

// 解析SSH服务地址 (user@host:port)
func parseSSHAddress(sshAddress string) (string, string, error) {
	parts := strings.Split(sshAddress, "@")
	if len(parts) != 2 {
		return "", "", errors.New(SSHAddressFormatError)
	}
	return parts[0], parts[1], nil
}

// 解析认证信息，分离密钥路径和密码
func parseAuthentication(auth string) (string, string) {
	if strings.HasPrefix(auth, "key:") {
		return strings.TrimPrefix(auth, "key:"), "" // 返回密钥路径
	}
	return "", auth // 返回密码
}

// 主函数
func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
