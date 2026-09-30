// Package hateoas implementa o motor HATEOAS do gateway: toda resposta recebe
// um nó "_links" cujos métodos e URLs dependem do estado de saúde da malha e
// do estado do Circuit Breaker.
package hateoas

import "net/http"

// Link descreve uma ação disponível para o cliente.
type Link struct {
	Href   string `json:"href" example:"/api/v1/services/payment"`
	Method string `json:"method" example:"GET"`
	Title  string `json:"title" example:"Atualizar status"`
	// Type indica o media type quando o recurso não é JSON (ex.: text/html).
	Type string `json:"type,omitempty"`
	// Body é o payload sugerido para a ação (POST/PUT).
	Body interface{} `json:"body,omitempty" swaggertype:"object"`
}

// Links é o mapa rel -> Link serializado como "_links".
type Links map[string]Link

// Resource deve ser embutido em toda resposta da API.
type Resource struct {
	Links Links `json:"_links"`
}

// Merge devolve um novo mapa com os links de todos os conjuntos (os últimos têm prioridade).
func Merge(sets ...Links) Links {
	out := Links{}
	for _, set := range sets {
		for rel, link := range set {
			out[rel] = link
		}
	}
	return out
}

func get(href, title string) Link {
	return Link{Href: href, Method: http.MethodGet, Title: title}
}

func post(href, title string, body interface{}) Link {
	return Link{Href: href, Method: http.MethodPost, Title: title, Body: body}
}

func put(href, title string, body interface{}) Link {
	return Link{Href: href, Method: http.MethodPut, Title: title, Body: body}
}
