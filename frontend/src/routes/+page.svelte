<!-- YOU CAN DELETE EVERYTHING IN THIS PAGE -->
<script lang="ts">
	import {Test, Struct, GetTest} from '$lib/wailsjs/go/main/App.js'
	Test().then((res) => {
		blah = res
	})
	Struct().then((res) => {
		console.log(res.name)
	})
	GetTest().then((res) => {
		console.log(res)
	})
	let blah = 'blah'
</script>
<div class="container h-full mx-auto flex justify-center items-center">
	{blah}dfsdfsdf
	<input type="file">
	<div class="space-y-5">
		<h1 class="h1">Let's building a Ui!!</h1>
		<a href="/test">A test link!</a>
		<p>Start by exploring:</p>
		<ul>
			<li><code class="code">/src/routes/+layout.svelte</code> - barebones layout, the CSS import order is critical!</li>
			<li><code class="code">/src/app.postcss</code> - minimal css to make the page full screen, may not be relevant for your project</li>
			<li><code class="code">/src/routes/+page.svelte</code> - this page, you can replace the contents</li>
		</ul>
	</div>
</div>
