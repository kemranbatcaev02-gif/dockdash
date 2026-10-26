package model

// Информация о Docker Context (локальный хост или удаленный сервер ssh/tcp)
type DockerContext struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Endpoint    string `json:"endpoint"`
	IsCurrent   bool   `json:"is_current"`
}
