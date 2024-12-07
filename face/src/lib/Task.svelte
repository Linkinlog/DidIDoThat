<script lang="ts">
type Task = {
  name: string;
  description: string;
  interval: "hourly" | "daily" | "weekly" | "monthly" | "yearly";
  intervalsCompleted: number[];
}

function intervalToActivityText(interval: Task["interval"]) {
  switch (interval) {
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

let { task, totalIntervals }: {task: Task} = $props();
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
    {#each Array(totalIntervals).fill(0) as _, index}
      <div
        class="activity-box"
        class:completed={task.intervalsCompleted.includes(index)}
      ></div>
    {/each}
  </div>
  <p class="activity-text">
    {task.intervalsCompleted.length} of {totalIntervals} {intervalToActivityText(task.interval)} completed
  </p>
</div>
