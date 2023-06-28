const Home = {
	view: () => [
		m("p", `
			Lorem, ipsum dolor sit amet consectetur adipisicing elit.
			Veritatis quos inventore sunt odit, explicabo quae modi quaerat, nam neque.
			Voluptatibus tenetur molestiae dolorum ipsum ratione perferendis voluptatem aperiam, deserunt asperiores!
		`),
		m("p", `
			Lorem, ipsum dolor sit amet consectetur adipisicing elit.
			Veritatis quos inventore sunt odit, explicabo quae modi quaerat, nam neque.
			Voluptatibus tenetur molestiae dolorum ipsum ratione perferendis voluptatem aperiam, deserunt asperiores!
		`),
	],
}

function onsubmit (e) {
	e.preventDefault()
}

export default Home
