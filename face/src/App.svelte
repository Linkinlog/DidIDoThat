<script lang="ts">
import Task from './lib/Task.svelte';

let creatingTask = false;

let tasks: Task[] = [];

$: (async () => {
  const response = await fetch('/api/tasks');
  tasks = await response.json();
})();

function createTask(e: Event) {
  e.preventDefault();
  const form = e.target as HTMLFormElement;
  const [name, description, interval] = form.elements as any;
  creatingTask = false;

  tasks = [
    ...tasks,
    {
      name: name.value,
      description: description.value,
      interval: interval.value,
    }
  ];

  let url = new URL('/api/tasks', window.location.href);
  url.searchParams.append('name', name.value);
  url.searchParams.append('description', description.value);
  url.searchParams.append('interval', interval.value);

  fetch(url, { method: 'POST' });

  // reset values
  name.value = '';
  description.value = '';
  interval.value = '';
}
</script>

<main>
  <h1><i>Did I Do That?</i></h1>

  {#if creatingTask}
    <form on:submit|preventDefault={createTask} id="new-task-form">
      <input type="text" placeholder="Task name" required />
      <input type="text" placeholder="Task description" />
      <select name="interval" required >
	<option value="hourly">Hourly</option>
	<option value="daily">Daily</option>
	<option value="weekly">Weekly</option>
	<option value="monthly">Monthly</option>
	<option value="yearly">Yearly</option>
      </select>
      <button type="submit">Create task</button>
    </form>
  {:else}
    <div id="tasks-container">
      {#each tasks as task}
	<Task task={task} totalIntervals={30} />
      {/each}
      <div id="get-started-container" role="button" tabindex="0" on:click={() =>creatingTask = true}>
	<p>Click here to create a new task</p>
      </div>
    </div>
  {/if}
</main>

<style>

</style>
