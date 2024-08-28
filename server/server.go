package server

type Server interface {
	Serve()                     // 监听端口 + 注册用户 + 读写长链接
	register(interface{}) *User // 注册用户 (map中创建)
	deregister(*User)           // 注销用户 (map中删除)
	write(*User)                // 写长链接
	read(*User)                 // 读长链接
}
