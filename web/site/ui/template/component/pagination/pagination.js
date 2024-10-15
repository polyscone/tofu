import { onMount } from "{{.Asset.TagJSImport "/js/dom.js"}}"

onMount(`[data-is="com.pagination"]`, node => {
	const select = node.querySelector(".pagination__info select")
	const form = select.closest("form")

	select.addEventListener("change", () => form.submit())
})
