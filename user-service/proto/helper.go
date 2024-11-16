package proto

func (request *LoginRequest) IsValid() bool {
	return len(request.Name) > 0 && len(request.Password) > 0
}
