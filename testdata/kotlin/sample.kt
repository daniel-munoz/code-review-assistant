package com.example.sample

import java.time.Instant
import kotlin.math.max

/**
 * A simple calculator with configurable precision.
 */
class Calculator(private val precision: Int) {
    companion object {
        const val MAX_VALUE = 1000000
    }

    fun add(a: Int, b: Int): Int {
        return a + b
    }

    fun subtract(a: Int, b: Int): Int {
        return a - b
    }
}

object Logger {
    fun log(message: String) {
        println(message)
    }
}

interface Repository {
    fun save(item: String)
}

// Greets a person by name.
fun greet(name: String): String {
    return "Hello, $name!"
}

fun main() {
    val calc = Calculator(2)
    println(calc.add(1, 2))
    println(greet("World"))
    println(max(1, 2))
    println(Instant.now())
}

enum class Status {
    ACTIVE, BANNED;

    fun describe(): String {
        return name.lowercase()
    }
}
