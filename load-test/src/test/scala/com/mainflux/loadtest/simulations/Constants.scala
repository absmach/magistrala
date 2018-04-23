package com.mainflux.loadtest.simulations

object Constants {
  val ManagerUrl = System.getProperty("manager", "http://localhost:8180")
  val HttpAdapterUrl = System.getProperty("http", "http://localhost:8182")
  val RequestsPerSecond = Integer.getInteger("requests", 100)
}