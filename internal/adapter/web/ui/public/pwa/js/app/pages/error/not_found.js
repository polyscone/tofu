function NotFound () {
	return {
		view: () => m("p", "The page you were looking for could not be found."),
	}
}

platform.routes.errorNotFound = {
	pattern: "/:rest...",
	component: NotFound,
}
