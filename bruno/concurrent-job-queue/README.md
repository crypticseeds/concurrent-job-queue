## Bruno collection: `concurrent-job-queue`

### Prerequisites
- API server running at `http://localhost:8080`
- Bruno Desktop and/or Bruno CLI (`bru`) installed

### What’s covered
- **Smoke**: `GET /health`, `GET /metrics`, `POST /tasks`, `GET /tasks/:id`
- **Negative/edge**: invalid JSON on create, not-found task ID, empty body create
- **Concurrency-friendly**: data-driven task creation (CSV) so you can generate lots of tasks quickly

### Import into Bruno (GUI)
Open Bruno → **Collections** → **Open Collection** → select the folder:
`bruno/concurrent-job-queue`

Then pick environment **`local`** (defaults to `http://localhost:8080`).

### Data-driven runs (many tasks)
The request `Requests/Concurrency/01_create_task_data_driven.bru` is designed for the Runner with a CSV file.

- **CSV file**: `bruno/concurrent-job-queue/data/create-tasks.csv`
- **How** (GUI): Runner → “Run with Parameters” → choose CSV → select the file above → run the **Concurrency** folder (or just that request).

### Concurrency (parallel load)
Bruno CLI runs requests **sequentially** within a single process, so to generate parallel load you have two practical options:

- **Option A (GUI Runner)**: run the same request with many iterations and enable parallelism in the Runner UI (if available in your Bruno version).
- **Option B (CLI, multiple processes)**: launch multiple `bru run` processes at the same time.

Example (from the collection root, once your server is running):

```bash
cd bruno/concurrent-job-queue

# 10 parallel runners, each iterating over the CSV once
for i in {1..10}; do
  bru run --env local --csv-file-path ./data/create-tasks.csv > "./results-$i.txt" 2>&1 &
done
wait
```

### Troubleshooting
- `Get Task - by ID (from create)` depends on `taskId` from `Create Task - with payload`. If run alone, it may fail because `taskId` is unset.

