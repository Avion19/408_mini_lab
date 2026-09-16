# Canvas Assignment Tracker

Canvas Assignment Tracker is a read-only web application for students who
want one focused view of their upcoming work. It loads active courses from
Boise State Canvas, combines upcoming and future assignments, filters them by
a selectable look-ahead window, and presents due dates, point values, urgency,
and links back to Canvas.

The project uses the same Go, Echo, Templ, and Tailwind stack as the Hello
World assignment. It is intentionally small, but demonstrates authenticated
REST API requests, JSON parsing, pagination, input handling, and safe local
secret management.

## Features

- Loads all active courses for the authenticated Canvas student.
- Supports viewing assignments from every active course or one selected course.
- Provides 7-, 14-, 30-, and 90-day look-ahead windows.
- Combines Canvas `upcoming` and `future` assignment results without duplicates.
- Excludes assignments that are already overdue or do not have a valid due date.
- Sorts assignments chronologically and labels assignments due within two days.
- Displays points, due dates, course names, and links to the original Canvas page.
- Follows Canvas pagination links until every page has been retrieved.
- Shows readable messages for missing credentials, network failures, invalid
  JSON, rate limits, and Canvas API errors.

## Technology stack

- Go 1.22+
- Echo v4 for HTTP routing and the web server
- Templ for type-safe server-rendered HTML
- Tailwind-generated CSS for the interface
- Canvas LMS REST API for course and assignment data

## Requirements

Before running the application, install:

1. Go 1.22 or newer: <https://go.dev/dl/>
2. A Boise State Canvas student account
3. A Canvas API access token

Node.js is not required to run the committed application. The generated Templ
file and compiled stylesheet are included in the repository.

## Setup

### 1. Clone the repository

Open a terminal and run:

```bash
git clone https://github.com/Avion19/408_mini_lab.git
cd 408_mini_lab
```

### 2. Download Go dependencies

From the project directory, run:

```bash
go mod download
```

This downloads the Go packages listed in `go.mod`.

### 3. Create a Canvas API token

1. Sign in to Canvas with your student account.
2. Open **Account Settings** from the profile menu in the global navigation.
3. Scroll to **Approved Integrations**.
4. Select **+ New Access Token**.
5. Enter a descriptive purpose, such as `CS408 Assignment Tracker`.
6. Set an expiration date (such as a few months from now), then select **Generate Token**.
7. Copy the token immediately. Canvas displays it only once.

### 4. Create the local environment file

Copy the safe template:

```bash
cp .env.example .env
```

Open `.env` in a text editor and replace the placeholder value for
`CANVAS_API_TOKEN` with the token from Canvas. The `.env` file is intentionally
ignored by Git. Never commit or share it.

### 5. Start the application

Run:

```bash
go run .
```

Open <http://localhost:8080> in a browser. Choose a course and look-ahead
window, then select **Load assignments**.

### 6. Run the tests

To run the automated checks:

```bash
go test ./...
```

The tests use an in-memory HTTP transport, so they do not require a Canvas
token or a network connection.

## Configuration

The application reads configuration from `.env` and from shell environment
variables. Shell variables take precedence over values in `.env`.

| Variable | Required | Description | Default |
| --- | --- | --- | --- |
| `CANVAS_API_TOKEN` | Yes | Canvas bearer token for the student account. | None |
| `CANVAS_BASE_URL` | No | HTTPS URL for the Canvas instance. | `https://boisestatecanvas.instructure.com` |
| `PORT` | No | Local HTTP port. | `8080` |

For example, to use a different local port:

```bash
PORT=9090 go run .
```

## How to use the application

When the home page loads, the application first retrieves the student’s active
courses. The form then accepts two inputs:

- **Course:** all active courses or one selected course
- **Look ahead:** assignments due in the next 7, 14, 30, or 90 days

After the form is submitted, the application requests both Canvas assignment
buckets, merges duplicate results by assignment ID, and applies the selected
date window locally. An assignment is displayed only when it has a valid due
date that is greater than or equal to the current time and less than or equal
to the selected cutoff.

## Canvas API endpoints used

| Method | Endpoint | Purpose |
| --- | --- | --- |
| `GET` | `/api/v1/courses?enrollment_state=active&per_page=100` | Retrieves the authenticated student’s active courses. |
| `GET` | `/api/v1/courses/:id/assignments?bucket=upcoming&order_by=due_at&per_page=100` | Retrieves assignments Canvas classifies as upcoming for a selected course. |
| `GET` | `/api/v1/courses/:id/assignments?bucket=future&order_by=due_at&per_page=100` | Retrieves assignments Canvas classifies as future for a selected course. |

The application uses two distinct Canvas resources: courses and assignments.
The assignments resource is requested with both Canvas bucket filters because
an assignment that fits the student’s selected date window may appear in the
`future` bucket rather than the narrower `upcoming` bucket.

All list requests inspect the HTTP `Link` header for `rel="next"` and continue
requesting pages until Canvas does not provide another page. Results from the
two assignment requests are then deduplicated before local date filtering and
sorting.

## Authentication, errors, and security

The application sends the token only in an HTTP bearer authorization header to
the configured HTTPS Canvas host. It never renders the token in the page or
writes it to application output.

Errors are displayed in the web interface instead of as raw JSON. The app
handles missing or placeholder tokens, invalid or expired tokens, forbidden
requests, rate limiting, network failures, malformed JSON, invalid pagination
links, and invalid form selections. If one course or assignment bucket fails
while loading several requests, the successful results remain visible and a
warning identifies the failed request.

## Development

The editable page is [views/home.templ](views/home.templ). The generated
Templ output is [views/home_templ.go](views/home_templ.go), and the stylesheet
source is [static/css/input.css](static/css/input.css).

If the optional development tools are installed, the Makefile provides these
commands:

```bash
make generate  # Regenerate Templ Go output
make css       # Rebuild the committed CSS stylesheet
make test      # Regenerate templates and run tests
make run       # Regenerate templates and start the app
```

The normal `go run .` and `go test ./...` commands do not require Node.js or
the optional CSS tooling.

## Project structure

```text
.
├── canvas.go             # Canvas client, authentication, JSON parsing, pagination
├── main.go               # Echo route, form input, filtering, and view-model mapping
├── main_test.go          # Authentication, pagination, filtering, and error tests
├── views/
│   ├── home.templ        # Editable Templ page
│   ├── home_templ.go     # Generated Templ output
│   └── model.go          # Display data types and helpers
├── static/css/
│   ├── input.css          # CSS source
│   └── output.css         # Committed generated stylesheet
├── .env.example          # Safe configuration template
├── .gitignore             # Secrets and build-artifact exclusions
├── go.mod                 # Go module and dependency definitions
└── Makefile               # Optional development commands
```

## Reflection

This mini-lab made the difference between a static page and an API-backed
application concrete. I learned how Canvas bearer authentication works and how
to map only the useful fields from a large JSON response into a small display
model. Separating the Canvas client from the Echo handler also made the API
logic easier to reason about and test.

Pagination was what I first had trouble with. A first page can look
correct while silently hiding data, so the client follows Canvas `Link`
headers for every list request. Another challenge was understanding that
Canvas’s `upcoming` and `future` buckets are separate classifications. I tried running this
originally looking two weeks out and assignments I was expecting to show up weren't there.
It was then after finding out that upcoming and future are different that I was able
to incorporate the two to work how I originally envisioned. The app
now combines both buckets, removes duplicate assignments, and applies the
student-selected date window locally.

If I had more time, I would request Canvas submission information so the app
could distinguish an overdue assignment from one that has already been turned
in late. I would also add browser-level tests for the form flow, a small cache
for repeated refreshes so the website operates faster,
and calendar-event support for students who want to
export their deadlines.

## Demo GIF

The included recording shows the application starting and the form flow:

![Canvas Assignment Tracker demo](docs/demo.gif)
