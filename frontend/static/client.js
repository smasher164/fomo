main()

function main() {
	var div_create_event = document.querySelector(".div_create_event")
	div_create_event.children[0].addEventListener("click", RegisterOnClick("create_event"))
}

function RegisterOnClick(type) {
	switch (type) {
		case "create_event":
			var ce = {
				state: "create",
				toggle: function(ev) {
					if (ce.state == "create") {
						ce.state = "cancel"
						ev.target.insertAdjacentHTML("afterend", renderCreateEvent())
						document.querySelector(".btn_confirm_event")
							.addEventListener("click", RegisterOnClick("confirm_event"))
						ev.target.textContent = "Cancel"
					} else if (ce.state == "cancel") {
						ce.state = "create"
						ev.target.parentNode.removeChild(ev.target.nextElementSibling)
						ev.target.textContent = "Create Event"
					}
				}
			}
			return ce.toggle
		case "confirm_event":
			return function(ev) {
				ev.preventDefault()
				var form = ev.target.form
				var desc = form.querySelector("textarea[name='desc']").value
				var i = desc.indexOf('\n')
				var data = {
					type: "create_event",
					title: desc.slice(0,i),
					desc: desc.slice(i+1),
					// date: form.querySelector("input[name='date']").value,
					// time: form.querySelector("input[name='time']").value,
				}
				console.log(data)

				// console.log(form.querySelector("input[name='start']").value)

				// var xhr = new XMLHttpRequest()
				// xhr.open("POST", "/")
				// xhr.setRequestHeader("Content-Type", "application/json; charset=utf-8")
				// xhr.send(JSON.stringify(data))
			}
	}
}

function renderCreateEvent() {
	var s = "<form>"+
				"<textarea name='desc' placeholder='Title\nDescription'></textarea>"+
				"<input name='start' type='datetime-local'></input>"+
				"<input name='end' type='datetime-local'></input>"+
				// search for location
				"<button class='btn_confirm_event'>Confirm</button>"+
			"</form>"
			return s
}