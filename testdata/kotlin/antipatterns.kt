package com.example.antipatterns

import kotlinx.coroutines.runBlocking

fun tooManyParams(a: Int, b: Int, c: Int, d: Int, e: Int, f: Int, g: Int): Int {
    return a + b + c + d + e + f + g
}

fun manyReturns(x: Int): String {
    if (x < 0) return "negative"
    if (x == 0) return "zero"
    if (x < 10) return "small"
    if (x < 100) return "medium"
    return "large"
}

fun magicNumbers(count: Int): Int {
    val MAX_BATCH = 250
    val timeout = 4500
    return count * 86400 + timeout + MAX_BATCH
}

fun unsafeAccess(name: String?): Int {
    return name!!.length
}

fun blockingCall(): Int {
    return runBlocking {
        42
    }
}

fun deepNesting(items: List<Int>): Int {
    var total = 0
    for (i in items) {
        if (i > 0) {
            for (j in 0 until i) {
                if (j % 2 == 0) {
                    while (total < 50) {
                        total += 1
                    }
                }
            }
        }
    }
    return total
}

fun main() {
    runBlocking {
        println(blockingCall())
    }
}

class Runner {
    fun main() {
        runBlocking {
            println("not an entry point")
        }
    }
}
