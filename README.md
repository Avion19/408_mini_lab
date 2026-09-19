# Canvas Assignment Tracker

Canvas Assignment Tracker is a read-only web application that gives Boise
State students one focused view of their upcoming Canvas work. It loads active
courses, combines upcoming and future assignments, filters them by a selected
time window, and shows due dates, point values, urgency, and links back to
Canvas.

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

## Setup instructions

These instructions assume you have never used Go before. Go is both the
programming language and the toolchain used to build this project. Its built-in
package manager, called **Go modules**, reads `go.mod` and downloads the exact
library versions this application needs into Go's local package cache. You do
not need to install Node.js, npm, or any front-end build tools to run the
committed application.

### 1. Install Go 1.22 or newer

First, open a terminal and check whether Go is already installed:

```bash
go version
```

You need output beginning with `go version go1.22` or a newer version. If the
command is not found, or the version is older than 1.22, install Go using one
of these options:

| Operating system | Installation option |
| --- | --- |
| macOS | Download the macOS installer from [go.dev/dl](https://go.dev/dl/), or run `brew install go` if you use Homebrew. |
| Windows | Download and run the Windows MSI installer from [go.dev/dl](https://go.dev/dl/). Keep the default installation options. |
| Debian or Ubuntu | Run `sudo apt update` followed by `sudo apt install golang-go`. If `go version` is still older than 1.22, use the current Linux archive from [go.dev/dl](https://go.dev/dl/) instead. |
| Any operating system | Use the installer or archive instructions on [go.dev/dl](https://go.dev/dl/). |

After installing, close and reopen the terminal so its `PATH` is refreshed,
then run `go version` again. Do not continue until it reports Go 1.22 or newer.

### 2. Clone the repository

Choose a folder where you keep school projects, then run:

```bash
git clone https://github.com/Avion19/408_mini_lab.git
cd 408_mini_lab
```

The first command downloads a copy of the project. The second command moves
the terminal into that project folder; every remaining command in this guide
must be run there.

### 3. Download the Go dependencies

Run:

```bash
go mod download
```

This is Go's equivalent of an `npm install`: it reads `go.mod` and downloads
the Echo web framework and Templ libraries into your local Go module cache.
You normally run it once after cloning and again only when `go.mod` or `go.sum`
changes. Go may also download a missing dependency automatically when you run
the application.

### 4. Generate a Canvas access token

The application reads your own Canvas enrollments, so it needs a personal
Canvas API token. Sign in to Boise State Canvas in a browser, then:

1. Select **Account** in Canvas's left-hand global navigation.
2. Select **Settings**.
3. Scroll to the **Approved Integrations** section.
4. Select **+ New Access Token**.
5. For the purpose, enter something recognizable, such as `CS408 Assignment Tracker`.
6. Choose an expiration date, such as a few months from now, then select **Generate Token**.
7. Copy the value Canvas displays, which usually begins with `13~`.

Canvas shows the complete token only once. Store it long enough to put it in
the next step. If you close the dialog before copying it, generate a new token.
If your Canvas account does not show **+ New Access Token**, contact your
Canvas administrator; token creation may be disabled for that account.

### 5. Create the `.env` configuration file

Make your private configuration file from the included template:

```bash
cp .env.example .env
```

Open `.env` in a text editor. Replace only `your_canvas_token_here` with the
token you just copied. Leave the Boise State URL and port unchanged unless you
know you need different values:

```dotenv
CANVAS_API_TOKEN=13~pasteYourLongCanvasTokenHere
CANVAS_BASE_URL=https://boisestatecanvas.instructure.com
PORT=8080
```

Do not put the token in quotation marks or add spaces around the `=`. The token
is equivalent to a password for your Canvas account. `.env` is listed in
`.gitignore`, so Git will not include it in commits; never share it, commit it,
or paste it into an assignment submission. If it is exposed, revoke it under
**Approved Integrations** and create a replacement.

### 6. Run the application

Start the local web server:

```bash
go run .
```

When the terminal reports that Echo is listening on port `8080`, open
<http://localhost:8080> in a browser. The page loads your active courses. Pick
one course or **All active courses**, select a look-ahead window, and choose
**Load assignments**. Stop the server at any time by returning to the terminal
and pressing `Ctrl+C`.

### 7. Run the tests (optional)

Run the automated checks with:

```bash
go test ./...
```

The tests use an in-memory HTTP transport, so they do not need your Canvas
token or an internet connection.

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
application concrete. I learned how a Canvas bearer token becomes an
`Authorization` header, how to decode only the fields needed from a large JSON
response, and how keeping the Canvas client separate from the Echo handler
makes the API code easier to understand and test. Building the display model
also showed me that data returned by an API is not automatically ready for a
user interface; dates, missing values, and course names all need deliberate
handling.

Pagination was the most challenging part. The first response from Canvas can
look complete even when additional courses or assignments are hidden behind a
`Link` header, so the client follows every `rel="next"` link before using the
results. I also assumed that assignments in a two-week window would
all arrive in Canvas's `upcoming` bucket. Learning that `upcoming` and
`future` are separate classifications explained why expected assignments were
missing. I felt like that wasn't super intuitive but it makes sense.
Combining both buckets, removing duplicates by assignment ID, and
then applying the selected date window locally produced the behavior I had
in mind originally.

With more time, I would request Canvas submission data so the tracker could
identify work already submitted late instead of treating all past-due work the
same. I would add browser-level tests for the course and time-window form,
cache course data between refreshes, and support Canvas calendar events for
students who want to export deadlines. For a version intended for students
beyond this lab, I would replace personal tokens in `.env` with a Canvas OAuth
flow so users never need to give a deployed application a long-lived token.

## Demo GIF

The included recording shows the application starting and the form flow:

![Canvas Assignment Tracker demo](docs/demo.gif)
