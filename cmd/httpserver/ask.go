package httpserver

// import (
// 	"github.com/j32u4ukh/gos/server/ghttp"
// )

// type AskerManager struct {
// 	asker      *ghttp.HttpAsker
// 	contextMap map[int32]*ghttp.HttpContext
// }

// func NewAskerManager() *AskerManager {
// 	return &AskerManager{
// 		contextMap: make(map[int32]*ghttp.HttpContext),
// 	}
// }

// func (m *AskerManager) Init(ip string, port int32) error {
// 	m.asker = ghttp.NewHttpAsker(ip, port)
// 	// err := m.asker.Init(nil, nil)
// 	// if err != nil {
// 	// 	return errors.Wrap(err, "Failed to init http server")
// 	// }
// 	return nil
// }
