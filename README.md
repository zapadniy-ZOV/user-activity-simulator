# user-activity-simulator

Micro-service for personal pet-project, that uses unitdb as a storage

## API Documentation

### `/start`

1. **Route:** `POST /start`
2. **Args (Body):**

    ```json
    {
      "user_ids": ["string", "string", ...]
    }
    ```

    * `user_ids`: A list of strings representing the user IDs for which to start a movement simulation. This list cannot be empty.
3. **Answer:**
    * **On Success (200 OK):**
        * Content-Type: `text/plain`
        * Body: `Simulation started for {count} users. Will run for approximately {duration}.` (e.g., "Simulation started for 2 users. Will run for approximately 30s.")
    * **On Error (400 Bad Request):**
        * Content-Type: `text/plain`
        * Body examples:
            * `Invalid request body: {error_message}`
            * `User ID list cannot be empty`
    * **On Error (405 Method Not Allowed):**
        * Content-Type: `text/plain`
        * Body: `Method Not Allowed`

### `/stop`

1. **Route:** `POST /stop`
2. **Args:** None
3. **Answer:**
    * **On Success (200 OK):**
        * Content-Type: `text/plain`
        * Body: `All active simulations stopped.`
    * **On Error (405 Method Not Allowed):**
        * Content-Type: `text/plain`
        * Body: `Method Not Allowed`

### `/user/{user_id}`

1. **Route:** `GET /user/{user_id}`
2. **Args (Path):**
    * `user_id` (string): The ID of the user whose location data is being requested. This is a required part of the URL path.
3. **Args (Query - Optional):**
    * `min` (float): A float between 0.0 and 1.0 (inclusive) representing the starting percentage of the user's data to fetch (oldest data is 0.0). Defaults to 0.0 if not provided.
    * `max` (float): A float between 0.0 and 1.0 (inclusive) representing the ending percentage of the user's data to fetch (newest data is 1.0). Defaults to 1.0 if not provided. Must be greater than or equal to `min`.
4. **Answer:**
    * **On Success (200 OK):**
        * Content-Type: `application/json`
        * Body:

            ```json
            {
              "user_id": "string",
              "data": [
                {
                  "dx": float64,
                  "dy": float64,
                  "ts": "YYYY-MM-DDTHH:MM:SSZ" // ISO8601 Timestamp
                },
                // ... more LocationData objects
              ]
            }
            ```

            * `user_id`: The ID of the user.
            * `data`: A list of `LocationData` objects, where each object represents a recorded movement. This list may be filtered based on `min` and `max` query parameters.
                * `dx`: Change in X coordinate.
                * `dy`: Change in Y coordinate.
                * `ts`: Timestamp of the movement record in ISO8601 format.
    * **On Error (400 Bad Request):**
        * Content-Type: `text/plain`
        * Body examples:
            * `Invalid URL path. Expected /user/{user_id}`
            * `Invalid 'min' parameter. Must be a float between 0.0 and 1.0.`
            * `Invalid 'max' parameter. Must be a float between 0.0 and 1.0.`
            * `'min' parameter cannot be greater than 'max' parameter.`
    * **On Error (404 Not Found):**
        * Content-Type: `text/plain`
        * Body: `No data found for user {user_id} within the specified range`
    * **On Error (405 Method Not Allowed):**
        * Content-Type: `text/plain`
        * Body: `Method Not Allowed`
    * **On Error (500 Internal Server Error):**
        * Content-Type: `text/plain`
        * Body examples:
            * `Failed to retrieve data for user {user_id}: {error_message}`
            * `Failed to encode response`
