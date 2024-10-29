function NotFound () {
	return {
		view: () => m("p", "{{.T "pwa.page.error.not_found"}}"),
	}
}

platform.routes.register("/:rest...", {
	render: NotFound,
})
