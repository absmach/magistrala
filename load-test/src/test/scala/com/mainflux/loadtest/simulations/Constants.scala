package com.mainflux.loadtest.simulations

object Constants {
  val UsersUrl = System.getProperty("users", "http://localhost:8180")
  val ClientsUrl = System.getProperty("clients", "http://localhost:8182")
  val HttpAdapterUrl = System.getProperty("http", "http://localhost:8185")
  val RequestsPerSecond = Integer.getInteger("requests", 100)
}