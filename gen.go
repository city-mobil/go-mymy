package mymy

//go:generate mockgen -destination mock/handler_mock.go -package mymy_mock github.com/city-mobil/go-mymy/pkg/mymy EventHandler
