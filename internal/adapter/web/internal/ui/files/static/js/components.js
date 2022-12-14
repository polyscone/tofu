onMount('input, textarea', node => {
  // Set a data-invalid attribute on forms when they're submitted with
  // invalid inputs
  //
  // This is to allow for styling invalid form elements after submittal
  // in a more persistent way than is allowed with CSS only
  node.addEventListener('invalid', e => {
    const form = e.target.closest('form')

    if (form) {
      form.dataset.invalid = true
    }
  })
})
