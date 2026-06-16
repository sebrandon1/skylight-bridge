# Event Types

| Event | Description |
|---|---|
| `chore.completed` | A chore's status changed from pending to completed |
| `chore.uncompleted` | A chore's status changed from completed back to pending |
| `chore.all_completed` | All chores for a given kid on today's date are completed (fires once per kid per day) |
| `reward.redeemed` | A reward was redeemed |

## Event Data Fields

Each event carries a `data` map with fields you can use in templates and filters:

| Event | Field | Example value |
|---|---|---|
| `chore.completed` | `chore_title` | `"Clean room"` |
| `chore.completed` | `assignee_name` | `"Alice"` |
| `chore.completed` | `category_id` | `"cat-abc123"` |
| `chore.uncompleted` | `chore_title` | `"Clean room"` |
| `chore.uncompleted` | `assignee_name` | `"Alice"` |
| `chore.uncompleted` | `category_id` | `"cat-abc123"` |
| `chore.all_completed` | `assignee_name` | `"Alice"` |
| `chore.all_completed` | `category_id` | `"cat-abc123"` |
| `reward.redeemed` | `reward_title` | `"Invest $20 in VOO"` |
| `reward.redeemed` | `points` | `20` |
