function NotFound () {
	return {
		view: () => m("p", "The page you were looking for could not be found."),
	}
}

platform.routes.register("/:rest...", NotFound)
