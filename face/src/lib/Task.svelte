<script lang="ts">
import moment from 'moment';

let { task, totalIntervals }: {task: Task} = $props();

type Task = {
  id: number;
  name: string;
  description: string;
  interval: "hourly" | "daily" | "weekly" | "monthly" | "yearly";
  intervals_map: Map<Date, boolean>;
}

function intervals_completed(task: Task) {
  return Object.values(task.intervals_map).filter(Boolean).length;
}

function intervalToActivityText(interval: Task["interval"]) {
  const temp = interval.toLowerCase();
  switch (temp) {
    case "hourly":
      return "hours";
    case "daily":
      return "days";
    case "weekly":
      return "weeks";
    case "monthly":
      return "months";
    case "yearly":
      return "years";
  }
}

function toLocalTime(date: string): string {
  let format = "YYYY-MM-DDTHHZZZ";
  let parsedDate = moment(date,format).format();
  return new Date(parsedDate).toLocaleString();
}

function isUpToDate(task: Task): boolean{
  if (!task.intervals_map) {
    return false;
  }

  return task.intervals_map[mostRecentDate(task)];
}

function mostRecentDate(task: Task): string {
  const keys = Object.keys(task.intervals_map);
  if (keys.length === 0) {
    return "";
  }
  const mostRecentKey = keys.reduce((latest, current) =>
    latest > current ? latest : current
  );

  return mostRecentKey;
}

function activityTextFormatted(task: Task): string {
  return `${intervals_completed(task)} of ${totalIntervals} ${intervalToActivityText(task.interval)} completed`
}

function completeTask(task: Task) {
  const mostRecentKey = Object.keys(task.intervals_map).reduce((latest, current) =>
    latest > current ? latest : current
  );

  if (task.intervals_map[mostRecentKey]) {
    return;
  }

  let url = new URL(`/api/tasks/${task.id}/complete`, window.location.href);
  fetch(url, { method: 'POST' });


  task.intervals_map[mostRecentKey] = true;
  let taskElement = document.querySelector(`.task[data-id="${task.id}"]`);

  let interval = taskElement.querySelector(`.task-activity > div:last-of-type`);

  let activityText = taskElement.querySelector(".activity-text");

  let completeBtn = taskElement.querySelector(".complete-button");

  interval.classList.add("completed");
  activityText.textContent = activityTextFormatted(task);
  completeBtn.classList.add("hidden");
}

</script>

<div class="task" data-id={task.id}>
  {#if !isUpToDate(task)}
    <button class="complete-button" onclick={() => completeTask(task)}>Complete</button>
  {/if}
  <div class="task-details">
    <div class="task-header">
      <div class="name">
        <p>{task.name}</p>
      </div>
      <div class="interval">
        <p>{task.interval}</p>
      </div>
    </div>
  </div>
  <div class="task-activity">
    {#each Object.entries(task.intervals_map) as [date, completed]}
      <div
        class="activity-box tooltip"
        class:completed={completed}
      >
        <span class="tooltiptext">{toLocalTime(date)}</span>
      </div>
    {/each}
  </div>
  <p class="activity-text">
    {activityTextFormatted(task)}
  </p>
</div>
