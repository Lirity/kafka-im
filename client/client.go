package client

type Client interface {
	Connect()
	Disconnect()
	read(interface{})
	write(interface{})
}
