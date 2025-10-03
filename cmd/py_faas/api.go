package main

type Server struct {
	Unimplemented
}

var _ ServerInterface = (*Server)(nil)

// func (s *Server) GetUserMcpServices(w http.ResponseWriter, r *http.Request, params GetUserMcpServicesParams) {
// 	w.WriteHeader(http.StatusOK)
// }

// func (s *Server) ProcessFile(w http.ResponseWriter, r *http.Request) {
// 	w.WriteHeader(http.StatusOK)
// }

// func (s *Server) ProxyGetRequest(w http.ResponseWriter, r *http.Request, serviceUUID openapi_types.UUID) {
// 	w.WriteHeader(http.StatusOK)
// }

// func (s *Server) ProxyPostRequest(w http.ResponseWriter, r *http.Request, serviceUUID openapi_types.UUID) {
// 	w.WriteHeader(http.StatusOK)
// }
