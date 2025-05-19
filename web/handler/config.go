package handler

type RateLimitConfig struct {
	Capacity  float64
	Replenish float64
}

type RouterConfig struct {
	RateLimit RateLimitConfig
}

type Routers struct {
	Site  RouterConfig
	PWA   RouterConfig
	APIv1 RouterConfig
}
