package gifrender

// Service is a small composition root for the GIF renderer and render metadata
// store. API handlers keep source-task validation close to HTTP/auth logic while
// this package owns FFmpeg rendering and render persistence.
type Service struct {
	Renderer Renderer
	Store    *Store
}

func NewService(renderer Renderer, store *Store) *Service {
	return &Service{Renderer: renderer, Store: store}
}
