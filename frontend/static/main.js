main()

function main() {
	var btn_fb_signup = document.querySelector(".btn_fb_signup")
	btn_fb_signup.addEventListener("click", RegisterOnClick("facebook"))
}

function redirect(relative, origin) {
	if (origin == undefined) {
		location.href = location.origin + relative
	} else {
		location.href = origin + relative
	}
}

function RegisterOnClick(method) {
	return function(ev) {
		var auth = "/auth?method=" + encodeURIComponent(method)
		redirect(auth, "http://localhost:8084")
	}
}

