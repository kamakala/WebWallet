### Web Wallet: A Go-Based Portfolio Tracker

This project is a modern web application designed for tracking and visualizing personal investment portfolios. Built with a focus on simplicity and performance, it leverages a server-side rendering approach to deliver a seamless user experience.

-----

### Features

  - **Portfolio Management:** Add, track, and manage various investment assets (stocks, crypto, etc.).
  - **Dynamic Visualizations:** View your portfolio composition and asset distribution with interactive charts. The application uses server-side rendering to generate charts, eliminating the need for complex client-side JavaScript.
  - **Theme Switching:** Instantly switch between light and dark themes with persistent user settings, powered by Go's `context.Context` and HTTP cookies to prevent screen flicker.
  - **HTMX Integration:** Enjoy a dynamic single-page application feel without writing extensive JavaScript. HTMX handles all data filtering and content updates.
  - **Robust Backend:** A clean and scalable backend architecture built in Go.

-----

### Technologies Used

  * **Go:** The core backend language.
  * **templ:** A powerful HTML templating engine that allows for server-side component-based rendering.
  * **go-echarts:** A Go library for generating highly customizable and interactive charts.
  * **htmx:** A library that allows for dynamic HTML interactions using simple attributes, minimizing JavaScript complexity.
  * **net/http:** Go's standard library for building web servers.

-----

### Installation and Setup

To get the application up and running on your local machine, follow these steps.

#### Prerequisites

  * **Go:** You need to have Go installed (version 1.18 or higher is recommended).
  * **templ:** The `templ` CLI is required for compiling templates. You can install it with `go install github.com/a-h/templ/cmd/templ@latest`.
  * **Docker:** Make sure you have Docker installed and running on your system. You can download it from the [official Docker website](https://www.docker.com/).

Running a MongoDB instance inside a Docker container is a standard and efficient way to manage your database for development. It isolates the database environment and makes it easy to set up and tear down.


-----

#### Steps

1.  **Clone the repository:**

    ```bash
    git clone https://github.com/your-username/your-repo-name.git
    cd your-repo-name
    ```

2.  **Install Go dependencies:**

    ```bash
    go mod tidy
    ```

3.  **Generate Go template files:** The `templ` files need to be compiled into Go code.

    ```bash
    templ generate
    ```
-----
##### Database Setup

This project uses **MongoDB** as its database. For local development, the recommended way to run the database is by using Docker. This ensures a consistent environment without needing to install MongoDB directly on your host machine.

###### Running MongoDB with Docker

1.  **Start the MongoDB container:** Open your terminal and run the following command. This command pulls the official MongoDB image and starts a new container.  It maps the container's default port (`27017`) to the same port on your local machine, allowing your Go application to connect to it. Run where docker compose file is placed:

    ```bash
    docker compose up -d
    ```
You may need to run is with *sudo*

2.  **Verify the container is running:** You can check the status and id of your container with the `docker ps` command.

    ```bash
    docker ps
    ```


3.  **Connect your application:** Your Go application can now connect to the MongoDB instance using the host `localhost` and port `27017`.

#### Stopping the MongoDB container

When you're finished with your development session, you can stop and remove the container to free up resources.

1.  **Stop the container:**

    ```bash
    docker stop my-mongo-db-id
    ```

2.  **Remove the container:**

    ```bash
    docker rm my-mongo-db-id
    ```

After setting up the database you can finally run the application
------

4.  **Run the application:**

    ```bash
    go run ./cmd/web/main.go
    ```
    or using the Make file
    
    ```bash
    make run
    ```

The application will be accessible at `http://localhost:8080`.

-----

### Usage 🗺️

Once the application is running, you can:

  * Navigate to the home page to see your portfolio overview.
  * Use the visualization page to explore different chart types and data filters.
  * Click the theme toggle button to switch between light and dark modes.

-----

### Contributing

Contributions are welcome\! If you find a bug or have an idea for a new feature, please open an issue or submit a pull request.

-----

### License

This project is licensed under the MIT License. See the `LICENSE` file for more details.
