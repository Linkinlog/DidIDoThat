<script lang="ts">
let { task, totalIntervals }: {task: Task} = $props();

type Task = {
  name: string;
  description: string;
  interval: "hourly" | "daily" | "weekly" | "monthly" | "yearly";
  intervals_map: Map<Date, boolean>;
}

let intervals_completed = Object.values(task.intervals_map).filter(Boolean);

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

function toLocalTime(date: string) {
	return new Date(date).toLocaleString();
}

</script>

<div class="task">
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
    {intervals_completed.length} of {totalIntervals} {intervalToActivityText(task.interval)} completed
  </p>
</div>
