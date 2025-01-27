<h1>PowerTop Monitoring</h1>
<h3>Using Prometheus and Grafana</h3>

<p>PowerTOP is a terminal-based diagnostic tool that helps you monitor power usage by programs running on a Linux system when it is not plugged into a power source, making it suitable for unreliable power sources. For PowerTop to work easily in edge devices, an image is required, which is built with the help of a Docker image and is available at <a href="https://hub.docker.com/">DockerHub registry</a>.</p>
<p>Furthermore, the stats can be acquired with the help of Prometheus metrics and can be stored in the internal Prometheus TSDB. This data can be really helpful for alert management or even a visual representation of the stats using tools like Grafana.</p>

<p>While running thousands of applications in edge devices, monitoring and optimizing power consumption is crucial.</p>

<h3>Local Setup</h3>
<h4>Prerequisites</h4>
<ol>
   <li>Linux environment for running without a container<ul>
</ol>

<h3>Dev Setup</h3>

For this, PowerTop needs to be pre-installed.

Open up a terminal.

1. Clone the repo.

2. Go into the folder <code>cd powertop_monitoring</code>.

3. Run using the Go compiler <code>sudo go run cmd/main.go</code>.  
   PowerTop requires sudo permission to access the system stats.

4. Bare Prometheus metrics can be seen using <code>curl --silent 0.0.0.0:8887/metrics |egrep 'powertop|rpi'</code>.

<h3>Running Using Docker</h3>

1. For this, you need to use the --privileged flag, which would give it access to host energy stats:  
   <code>docker run -d -p 8887:8887 --privileged sibseh/powertopcsv:v2</code>.  
2. Bare Prometheus metrics can be seen using <code>curl 0.0.0.0:8887/metrics | grep powertop</code>.

These can be run with Grafana and Prometheus easily with the Docker Compose file.

<h3>Monitoring with Grafana and Prometheus using Docker Compose</h3>

1. Open up a terminal in the same directory <code>docker-compose up</code>.

2. Open your favorite browser at localhost:3000; it will open up Grafana. Log in with the username and password both as <code>admin</code>.

3. Go to Configuration -> Data Sources -> Add Prometheus -> set HTTP as <code>http://prometheus:9090</code>.

4. Go to Create -> Dashboard -> Select one.

5. Add powertop_wakeup_count, powertop_baseline_power, powertop_tunable_count, powertop_cpu_usage.

6. Now you can clearly see the parameters of your system calculated!

The final setup should look like this:
![Screenshot from 2022-09-02 05-41-50](https://user-images.githubusercontent.com/95071627/187992443-a1e15061-577f-408c-a9ba-ad520a47cbb9.png)

Viewing powertop_baseline_power variation.

NOTE:  
On amd64 systems we provide results from vcgencmd : 
```
power_metrics_exporter_build_info A metric with a constant '1' value labeled by version, revision, branch, goversion from which power_metrics_exporter was built, and the goos and goarch for the build.
 powertop_baseline_power_count counts the baseline power used available by powertop
 powertop_cpu_usage_count counts the cpu usage in % by powertop
 powertop_tunables_count counts the number of tuning available by powertop
 powertop_wakeup_count counts the wake up calls per second available by powertop
 process_cpu_seconds_total Total user and system CPU time spent in seconds.
 process_max_fds Maximum number of open file descriptors.
 process_open_fds Number of open file descriptors.
 process_resident_memory_bytes Resident memory size in bytes.
 process_start_time_seconds Start time of the process since unix epoch in seconds.
 process_virtual_memory_bytes Virtual memory size in bytes.
 process_virtual_memory_max_bytes Maximum amount of virtual memory available in bytes.
 promhttp_metric_handler_requests_in_flight Current number of scrapes being served.
 promhttp_metric_handler_requests_total Total number of scrapes by HTTP status code.
```
On arm64 systems we provide these results from powertop:
```
 rpi_clock_frequency_hz Clock frequencies for different components
 rpi_cpu_frequency_mhz CPU frequency in MHz
 rpi_cpu_temperature_celsius CPU temperature in Celsius
 rpi_memory_split_mb Memory split between ARM and GPU in MB
 rpi_power_kilowatts Current power usage in kilowatts
 rpi_power_kwh_per_day Estimated kilowatt hours per day
 rpi_power_kwh_per_month Estimated kilowatt hours per month
 rpi_power_state Power state indicators (1 for true, 0 for false)
 rpi_power_watts Current power usage in watts
 rpi_voltage_core Core voltage in volts
 rpi_voltage_sdram SDRAM voltage readings
```
 
Use the stable tag from Docker Hub; it is the bullseye version.
