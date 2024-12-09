<script lang="ts">
import Task from './lib/Task.svelte';
import QRCode from 'qrcode';

let creatingTask = false;
let showLogin = false;
let showProfile = false;
let loggedIn = false;

let tasks: Task[] = [];

$: (async () => {
  const response = await fetch('/api/tasks');
  tasks = await response.json();
})();

$: (async () => {
  const response = await fetch('/api/auth/session');
  if (response.status >= 300) {
    return;
  }
  const { username } = await response.json();
  localStorage.setItem('username', username);
  loggedIn = true;
})();

function createTask(e: Event) {
  e.preventDefault();
  const form = e.target as HTMLFormElement;
  const [name, description, interval] = form.elements as any;

  tasks = [
    ...tasks,
    {
      name: name.value,
      description: description.value,
      interval: interval.value,
      intervals_map: new Map()
    }
  ];

  let url = new URL('/api/tasks', window.location.href);
  url.searchParams.append('name', name.value);
  url.searchParams.append('description', description.value);
  url.searchParams.append('interval', interval.value);

  fetch(url, { method: 'POST' });

  name.value = '';
  description.value = '';
  interval.value = '';

  window.location.reload();
}

async function login(e: Event) {
  e.preventDefault();
  const form = e.target as HTMLFormElement;
  const [username, password] = form.elements as any;

  const response = await fetch('/api/auth/login', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({
      username: username.value,
      password: password.value
    })
  });

  if (response.status < 300) {
    window.location.reload();
  } else {
    const toast = document.getElementById('toast');
    toast.textContent = await response.text();
    toast.classList.add('error');
    toast.classList.remove('hidden');
    setTimeout(() => {
      toast.classList.add('hidden');
    }, 3000);
  }
}

function logout() {
  localStorage.removeItem('username');
  loggedIn = false;
  fetch('/api/auth/logout')
  window.location.reload();
}

async function getLoginQR(e: Event) {
  let url = new URL('/api/auth/qr', window.location.href);
  const response = await fetch(url);

  if (response.status >= 300) {
    return;
  }

  const qrURL = await response.text();

  const fullQrURL = new URL(`/api/auth/magic/${qrURL}`, window.location.href);
  
  const profilePage = document.getElementById('profile-page');

  const anchor = document.createElement('a');
  anchor.href = fullQrURL.href;
  anchor.textContent = 'Or copy this link to login';
  anchor.id = 'qr-link';

  if (!document.getElementById('qr-link')) {
    profilePage.appendChild(anchor);
  }

  const qrCanvas = document.getElementById('qr-code') as HTMLCanvasElement;
  qrCanvas.height = 600;
  qrCanvas.width = 200;
  const qrContext = qrCanvas.getContext('2d');

  QRCode.toCanvas(qrCanvas, fullQrURL.href);

  (e.target as HTMLButtonElement).disabled = true;
}
</script>

<main>
  <nav id="main-nav">
    {#if loggedIn}
      <button
	class="nav-link"
	onclick={() => showProfile = !showProfile}
      >Profile</button>
      <button
	class="nav-link"
	onclick={logout}
      >Logout</button>
    {:else}
      <button
	class="nav-link"
	onclick={() => showLogin = !showLogin}
      >Login</button>
    {/if}
  </nav>
  <div id="toast" class="hidden"></div>

  <h1><a href="/"><i>Did I Do That?</i></a></h1>

  {#if showLogin}
    <p>Log in or sign up here</p>
    <form onsubmit={login} id="login-form">
      <input class="login-username" type="text" placeholder="Username" required />
      <input class="login-password" type="password" placeholder="Password" required />
      <button type="submit">Login</button>
    </form>
  {:else}
    {#if showProfile}
      <div id="profile-page">
	<p>Username: {localStorage.getItem('username')}</p>
	<p>Login quickly by scanning the QR code below</p>
	<button
	  class="get-qr-btn"
	  onclick={(e) => getLoginQR(e)}
	>Get QR code</button>
	<canvas id="qr-code"></canvas>
      </div>
    {:else}
      {#if !loggedIn}
	<p>Log in to see your tasks</p>
      {:else}
	{#if creatingTask}
	  <form onsubmit={createTask} id="new-task-form">
	    <input class="input-field" type="text" placeholder="Task name" required />
	    <input class="input-field" type="text" placeholder="Task description" />
	    <label for="interval">Interval</label>
	    <select class="input-field"  name="interval" required >
	      <option value="hourly">Hourly</option>
	      <option value="daily">Daily</option>
	      <option value="weekly">Weekly</option>
	      <option value="monthly">Monthly</option>
	      <option value="yearly">Yearly</option>
	    </select>
	    <button class="create-task-btn" type="submit">Create task</button>
	  </form>
	{:else}
	  <div id="tasks-container">
	    <div id="get-started-container" role="button" tabindex="0" onclick={() =>creatingTask = true}>
	      <p>Click here to create a new task</p>
	    </div>
	    {#each tasks as task}
	      <Task task={task} totalIntervals={30} />
	    {/each}
	  </div>
	{/if}
      {/if}
    {/if}
  {/if}
</main>
