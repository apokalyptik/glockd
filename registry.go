package main

type registration_request struct {
	client string
	name string
	reply chan string
	dump chan map[string] string
}

var registrar = make(chan registration_request, 8192)

func mind_registry() {
	registry := map[string] string{}
	for {
		req := <-registrar
		if req.dump != nil {
			req.dump<- registry
		} else if req.reply != nil {
			v, present := registry[req.client]
			if present {
				req.reply<- v
			} else {
				req.reply<- req.client
			}
		} else if req.name == "" {
			_, present := registry[req.client]
			if present {
				delete(registry, req.client)
			}
		} else {
			registry[req.client] = req.name
		}
	}
}
