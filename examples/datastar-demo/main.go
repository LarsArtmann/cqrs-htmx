package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	cqrs := NewCQRS()

	mux := http.NewServeMux()

	mux.HandleFunc("GET /", serveIndex)
	mux.HandleFunc("POST /api/todos", handleCreateTodo(cqrs))
	mux.HandleFunc("POST /api/todos/toggle", handleToggleTodo(cqrs))
	mux.HandleFunc("POST /api/todos/delete", handleDeleteTodo(cqrs))
	mux.HandleFunc("POST /api/todos/update", handleUpdateTodo(cqrs))
	mux.HandleFunc("GET /api/todos", handleListTodos(cqrs))
	mux.HandleFunc("GET /api/events", handleEventStream(cqrs))
	mux.HandleFunc("GET /api/events/replay", handleEventReplay(cqrs))
	mux.HandleFunc("POST /api/simulate", handleSimulate(cqrs))

	addr := ":8095"
	fmt.Printf("CQRS + Datastar Todo Demo\n")
	fmt.Printf("Event-sourced: all mutations produce domain events\n")
	fmt.Printf("Real-time: open multiple tabs to see live event stream\n")
	fmt.Printf("Listening on http://localhost%s\n", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func serveIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(indexHTML))
}

const indexHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>CQRS + Datastar Todo Demo</title>
    <script type="module" src="https://cdn.jsdelivr.net/gh/starfederation/datastar@v1.0.1/bundles/datastar.js"></script>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            max-width: 900px;
            margin: 0 auto;
            padding: 2rem;
            background: #0d1117;
            color: #e6edf3;
        }
        h1 { margin-bottom: 0.5rem; font-size: 1.8rem; }
        .subtitle { color: #8b949e; margin-bottom: 2rem; font-size: 0.9rem; }
        .subtitle code { background: #161b22; padding: 0.2rem 0.4rem; border-radius: 4px; font-size: 0.85rem; }

        .layout {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 2rem;
            align-items: start;
        }
        .panel {
            background: #161b22;
            border: 1px solid #30363d;
            border-radius: 8px;
            padding: 1.5rem;
        }
        .panel h2 {
            font-size: 1rem;
            margin-bottom: 1rem;
            color: #8b949e;
            text-transform: uppercase;
            letter-spacing: 0.05em;
        }

        /* Notification */
        .notification {
            padding: 0.75rem 1rem;
            border-radius: 6px;
            margin-bottom: 1rem;
            font-size: 0.9rem;
            transition: opacity 0.3s;
        }
        .notification.success { background: #0f5323; border: 1px solid #238636; }
        .notification.error { background: #5a1a1a; border: 1px solid #da3633; }
        .notification.info { background: #0c2d6b; border: 1px solid #388bfd; }
        .notification.warning { background: #4a3000; border: 1px solid #d29922; }

        /* Form */
        .input-group {
            display: flex;
            gap: 0.5rem;
            margin-bottom: 1rem;
        }
        .input-group input {
            flex: 1;
            padding: 0.6rem 0.8rem;
            background: #0d1117;
            border: 1px solid #30363d;
            border-radius: 6px;
            color: #e6edf3;
            font-size: 0.95rem;
        }
        .input-group input:focus { border-color: #388bfd; outline: none; }
        .input-group button {
            padding: 0.6rem 1.2rem;
            background: #238636;
            color: white;
            border: none;
            border-radius: 6px;
            cursor: pointer;
            font-size: 0.95rem;
        }
        .input-group button:hover { background: #2ea043; }

        /* Stats */
        .stats {
            display: flex;
            gap: 1rem;
            margin-bottom: 1rem;
            font-size: 0.85rem;
            color: #8b949e;
        }
        .stats strong { color: #e6edf3; }

        /* Todo list */
        .todo-item {
            display: flex;
            align-items: center;
            gap: 0.75rem;
            padding: 0.6rem 0;
            border-bottom: 1px solid #21262d;
            list-style: none;
        }
        .todo-item:last-child { border-bottom: none; }
        .todo-item.completed .todo-title { text-decoration: line-through; color: #484f58; }
        .todo-title { flex: 1; }
        .delete-btn {
            background: none;
            border: 1px solid #30363d;
            color: #8b949e;
            border-radius: 4px;
            cursor: pointer;
            padding: 0.2rem 0.5rem;
            font-size: 0.8rem;
        }
        .delete-btn:hover { border-color: #da3633; color: #da3633; }
        .empty-state { color: #484f58; font-style: italic; padding: 1rem 0; list-style: none; }

        /* Event log */
        .event-entry {
            display: flex;
            gap: 0.75rem;
            padding: 0.4rem 0;
            border-bottom: 1px solid #21262d;
            font-family: "SF Mono", Monaco, monospace;
            font-size: 0.8rem;
        }
        .event-type {
            background: #1f6feb;
            color: white;
            padding: 0.1rem 0.4rem;
            border-radius: 4px;
            font-size: 0.75rem;
        }
        .event-agg { color: #8b949e; }
        .event-time { color: #484f58; margin-left: auto; }
        #event-log { max-height: 400px; overflow-y: auto; }

        /* Architecture labels */
        .arch-label {
            display: inline-block;
            font-size: 0.7rem;
            padding: 0.15rem 0.4rem;
            border-radius: 3px;
            margin-left: 0.5rem;
            vertical-align: middle;
        }
        .arch-label.write { background: #238636; color: white; }
        .arch-label.read { background: #1f6feb; color: white; }
        .arch-label.event { background: #8957e5; color: white; }

        /* Simulate button */
        .simulate-btn {
            width: 100%;
            padding: 0.6rem 1rem;
            background: #8957e5;
            color: white;
            border: none;
            border-radius: 6px;
            cursor: pointer;
            font-size: 0.9rem;
            margin-bottom: 0.5rem;
        }
        .simulate-btn:hover:not(:disabled) { background: #a371f7; }
        .simulate-btn:disabled { opacity: 0.5; cursor: not-allowed; }

        /* Event user badge */
        .event-user {
            background: #30363d;
            color: #e6edf3;
            padding: 0.1rem 0.4rem;
            border-radius: 4px;
            font-size: 0.75rem;
        }
    </style>
</head>
<body>
    <h1>CQRS + Datastar Todo Demo</h1>
    <p class="subtitle">
        Event-sourced with <code>go-cqrs-lite</code>.
        Frontend via <code>Datastar SSE</code> (no JS written by hand).
        Open multiple tabs to see real-time event streaming.
    </p>

    <!--
        Signals: Datastar's reactive client state.
        notification: {level, message} — patched by the server via SSE.
        title: input binding for new todo.
    -->
    <div
        data-signals="{notification: {level: '', message: ''}, title: '', simulating: false}"
        data-computed:hasNotification="$notification.message != ''"
    >
        <!-- Notification (reactive signal, updated by server) -->
        <div data-show="$hasNotification" class="notification" data-attr:class="'notification ' + $notification.level">
            <span data-text="$notification.message"></span>
        </div>

        <div class="layout">
            <!-- Left: Todo CRUD -->
            <div class="panel">
                <h2>Todos <span class="arch-label write">Command</span> <span class="arch-label read">Query</span></h2>

                <!--
                    Create form: data-bind:title sends the signal value.
                    @post('/api/todos') sends a POST with signals as JSON body.
                -->
                <form class="input-group" data-on:submit="@post('/api/todos')">
                    <input
                        type="text"
                        data-bind:title
                        placeholder="What needs to be done?"
                        required
                    >
                    <button type="submit">Add</button>
                </form>

                <!-- Stats (patched by server after each command) -->
                <div id="stats" class="stats"></div>

                <!--
                    Todo list: server patches items via SSE PatchElements.
                    On page load, @get fetches the initial list.
                -->
                <ul id="todo-list" data-on:load="@get('/api/todos')"></ul>
            </div>

            <!-- Right: Live Event Stream -->
            <div class="panel">
                <h2>Event Stream <span class="arch-label event">SSE</span></h2>
                <p style="color: #8b949e; font-size: 0.85rem; margin-bottom: 1rem;">
                    All domain events, streamed in real-time via SSE.
                    Click the button below to simulate other users.
                </p>

                <!-- Simulate button: sends signals {count} and starts background bots -->
                <button
                    class="simulate-btn"
                    data-on:click="@post('/api/simulate')"
                    data-indicator:_simulating
                    data-attr:disabled="$simulating || $_simulating"
                >
                    <span data-show="!$simulating && !$_simulating">Simulate 10 Users</span>
                    <span data-show="$simulating || $_simulating">Simulating...</span>
                </button>

                <div style="margin-top: 0.5rem; font-size: 0.8rem; color: #484f58;" data-show="$simulating">
                    Bots are creating, toggling, and deleting todos. Watch the event stream!
                </div>

                <!--
                    data-on:load subscribes to the SSE event stream.
                    The connection stays open — server pushes new events as they happen.
                -->
                <div style="margin-top: 1rem;">
                    <div id="event-log" data-on:load="@get('/api/events')"></div>
                </div>
            </div>
        </div>
    </div>
</body>
</html>`
