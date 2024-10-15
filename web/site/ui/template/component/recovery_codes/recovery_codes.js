import { onMount } from "{{.Asset.TagJSImport "/js/dom.js"}}"

onMount(`[data-is="com.recovery_codes"]`, node => {
	const codeList = node.querySelector(".recovery-code-list__codes")

	if (!codeList) {
		return
	}

	const actions = node.querySelector(".recovery-code-list__actions")

	if (codeList) {
		actions.style.display = ""
	}

	const codes = Array.from(codeList.querySelectorAll("li")).map(li => li.innerText.trim()).filter(code => !!code)
	const download = node.querySelector(".recovery-codes-download")
	const print = node.querySelector(".recovery-codes-print")
	const copy = node.querySelector(".recovery-codes-copy")

	if (download) {
		download.addEventListener("click", event => {
			event.preventDefault()

			let data = ""

			for (const code of codes) {
				data += `${code}\r\n`
			}

			const id = "download-recovery-codes-link"
			let link = document.getElementById(id)

			if (!link) {
				link = document.createElement("a")

				document.body.appendChild(link)
			}

			link.id = id
			link.setAttribute("href", `data:text/plain;charset=utf-8,${encodeURIComponent(data)}`)
			link.setAttribute("download", `recovery-codes-${window.location.hostname}.txt`)

			document.getElementById(id).click()
		})
	}

	if (print) {
		print.addEventListener("click", event => {
			event.preventDefault()

			const screenStyleId = "recovery-codes-screen-style"
			const printStyleId = "recovery-codes-print-style"
			const containerId = "recovery-codes-print-container"

			let screenStyle = document.getElementById(screenStyleId)

			if (!screenStyle) {
				screenStyle = document.createElement("style")

				screenStyle.media = "screen"
				screenStyle.id = screenStyleId
				screenStyle.innerHTML = `
					#${containerId} {
						display: none !important;
					}
				`

				document.body.appendChild(screenStyle)
			}

			let printStyle = document.getElementById(printStyleId)

			if (!printStyle) {
				printStyle = document.createElement("style")

				printStyle.media = "print"
				printStyle.id = printStyleId
				printStyle.innerHTML = `
					body * {
						display: none !important;
					}

					#${containerId},
					#${containerId} * {
						display: block !important;
						text-align: center;
					}

					#${containerId} ul {
						padding: 0;
						list-style: none;
						font-family: monospace;
						font-size: 2rem;
					}
				`

				document.body.appendChild(printStyle)
			}

			let printContainer = document.getElementById(containerId)

			if (!printContainer) {
				printContainer = document.createElement("div")
				printContainer.id = containerId

				document.body.appendChild(printContainer)
			}

			let data = ""

			for (const code of codes) {
				data += `<li>${code}</li>`
			}

			printContainer.innerHTML = `
				<h1>
					Two-factor authentication recovery codes<br>
					<small>${window.location.hostname}</small>
				</h1>
				<ul>${data}</ul>
			`

			setTimeout(() => {
				window.print()

				printContainer.parentNode.removeChild(printContainer)
				printStyle.parentNode.removeChild(printStyle)
				screenStyle.parentNode.removeChild(screenStyle)
			}, 0)
		})
	}

	if (copy) {
		copy.addEventListener("click", event => {
			event.preventDefault()

			let data = ""

			for (const code of codes) {
				data += `${code}\r\n`
			}

			const textarea = document.createElement("textarea")

			textarea.value = data
			textarea.style.top = "0"
			textarea.style.left = "0"
			textarea.style.position = "fixed"

			document.body.appendChild(textarea)

			textarea.focus()
			textarea.select()

			const result = document.execCommand("copy")

			if (result) {
				event.target.innerText = "Copied"
			} else {
				event.target.innerText = "Failed"
			}

			textarea.parentNode.removeChild(textarea)
		})
	}
})
