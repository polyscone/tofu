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

			return [
				vnode.attrs.label ? m("label", {
					for: id,
					class: vnode.attrs.required ? "required" : null,
				}, vnode.attrs.label) : null,
				m(`input[type=${vnode.attrs.type}]`, {
					oninput: vnode.attrs.oninput,
					value: vnode.attrs.value,
					id: id,
					required: vnode.attrs.required,
					placeholder: vnode.attrs.placeholder,
					autocomplete: vnode.attrs.autocomplete,
					pattern: vnode.attrs.pattern,
					minlength: vnode.attrs.minlength,
					maxlength: vnode.attrs.maxlength,
					class: vnode.attrs.error ? "invalid" : null,
				}),
				m("p.error", vnode.attrs.error),
			]
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