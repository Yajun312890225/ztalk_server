package server

//RouterMap 路由
type RouterMap struct {
	pools map[int16]func(uint32, map[string]interface{}) bool
}

//NewRouterMap router
func NewRouterMap() *RouterMap {
	return &RouterMap{
		pools: make(map[int16]func(uint32, map[string]interface{}) bool),
	}
}

//Register 注册
func (r *RouterMap) Register(cmdid int16, funcs func(uint32,map[string]interface{}) bool) bool {
	if _, exit := r.pools[cmdid]; !exit {
		r.pools[cmdid] = funcs
		return true
	}
	return false
}
