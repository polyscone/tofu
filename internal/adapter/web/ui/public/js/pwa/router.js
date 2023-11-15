import Layout from "./layout.js"
import Home from "./home.js"
import NotFound from "./not_found.js"

function handle (component) {
	return {
		onmatch (args, requestedPath) {
			if (m.route.get() === requestedPath) {
				return new Promise(() => {})
			}
		},
		render: () => m(Layout, m(component)),
	}
}

export default function () {
	m.route.prefix = platform.config.prefix

	m.route(document.body, "/", {
		"/": handle(Home),
		"/:rest...": handle(NotFound),
	})
}
