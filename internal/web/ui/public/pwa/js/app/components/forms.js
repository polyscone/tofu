export function Input () {
	return {
		view (vnode) {
			if (!vnode.attrs.type) {
				throw new Error("Form input requires a type")
			}
			if (!vnode.attrs.name) {
				throw new Error("Form input requires a name")
			}

			const id = vnode.attrs.id || `form__${vnode.attrs.name}`

			const input = [
				vnode.attrs.label ? m("label", {
					for: id,
					class: vnode.attrs.required ? "required" : null,
				}, vnode.attrs.label) : null,
				m(`input[type=${vnode.attrs.type}]`, {
					oninput: vnode.attrs.oninput,
					value: vnode.attrs.value,
					id: id,
					required: vnode.attrs.required,
					disabled: vnode.attrs.disabled,
					placeholder: vnode.attrs.placeholder,
					autocomplete: vnode.attrs.autocomplete,
					pattern: vnode.attrs.pattern,
					step: vnode.attrs.step,
					min: vnode.attrs.min,
					max: vnode.attrs.max,
					minlength: vnode.attrs.minlength,
					maxlength: vnode.attrs.maxlength,
					class: vnode.attrs.error ? "invalid" : null,
				}),
				m("p.error", vnode.attrs.error),
			]

			if (vnode.attrs.container) {
				return m(vnode.attrs.container, input)
			}

			return input
		},
	}
}

export function Select () {
	return {
		view (vnode) {
			if (!vnode.attrs.options) {
				throw new Error("Select requires an options array")
			}
			if (!vnode.attrs.name) {
				throw new Error("Select requires a name")
			}

			const value = String(vnode.attrs.value || "")
			const options = []

			if (vnode.attrs.placeholder) {
				options.push(m("option", { value: "", hidden: true, disabled: true, selected: !value }, vnode.attrs.placeholder))
			}

			for (const option of vnode.attrs.options) {
				const optionValue = String(option.value || option.label || "")

				options.push(m("option", { value: optionValue, selected: value && optionValue === value }, option.label))
			}

			const id = vnode.attrs.id || `form__${vnode.attrs.name}`
			const select = [
				vnode.attrs.label ? m("label", {
					for: id,
					class: vnode.attrs.required ? "required" : null,
				}, vnode.attrs.label) : null,
				m("select", {
					onchange: vnode.attrs.onchange,
					id: id,
					required: vnode.attrs.required,
					class: vnode.attrs.error ? "invalid" : null,
				}, options),
				m("p.error", vnode.attrs.error),
			]

			if (vnode.attrs.container) {
				return m(vnode.attrs.container, select)
			}

			return select
		},
	}
}

export function EmailInput () {
	return {
		view: vnode => m(Input, {
			...vnode.attrs,
			type: "email",
			placeholder: "email@example.com",
			pattern: "^[a-zA-Z0-9+._~\\-]+@[a-zA-Z0-9+._~\\-]+(\.[a-zA-Z0-9+._~\\-]+)+$",
			maxlength: 100,
		}),
	}
}

export function PasswordInput () {
	return {
		view: vnode => m(Input, {
			...vnode.attrs,
			type: "password",
			placeholder: "password or passphrase",
			pattern: "^.+$",
			minlength: 8,
			maxlength: 1000,
		}),
	}
}

export function TOTPInput () {
	return {
		view: vnode => m(Input, {
			...vnode.attrs,
			type: "text",
			placeholder: "123456",
			pattern: "^\\d+$",
			minlength: 6,
			maxlength: 6,
		}),
	}
}

export function RecoveryCodeInput () {
	return {
		view: vnode => m(Input, {
			...vnode.attrs,
			type: "text",
			placeholder: "ABCDEFGHIJKLM",
			pattern: "^[A-Z2-7]+$",
			minlength: 13,
			maxlength: 13,
		}),
	}
}

export function TokenInput () {
	return {
		view: vnode => m(Input, {
			...vnode.attrs,
			type: "text",
			placeholder: "ABCDEFGHIJKLM",
			pattern: "^[A-Z2-7]+$",
			minlength: 13,
			maxlength: 13,
		}),
	}
}
