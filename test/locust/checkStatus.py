from locust import HttpLocust, TaskSet, task, ResponseError

class CheckStatus(TaskSet):
    def on_start(self):
        """ on_start is called when a Locust start before any task is scheduled """
        print("Locus test started")

    @task(1)
    def getVersion(self):
        headers = {'Content-Type': 'application/json', 'Accept': 'application/json'}
        response = self.client.get("/version", headers=headers)

        print "Response status code:", response.status_code
        print "Response content:", response.content

class HelloLocust(HttpLocust):
    task_set = CheckStatus
    min_wait=5000
    max_wait=9000
