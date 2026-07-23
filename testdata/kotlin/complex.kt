package com.example.complex

fun highlyComplexFunction(a: Int, b: Int, c: Int, d: Boolean, e: String?, f: List<Int>): Int {
    var result = 0
    if (a > 10 && b > 20) {
        result += 1
    }
    if (c > 30 || d) {
        result += 2
    }
    val name = e ?: "unknown"
    when (a) {
        1 -> result += 10
        2 -> result += 20
        3 -> result += 30
        else -> result += 40
    }
    for (item in f) {
        if (item > 100) {
            result += item
        }
    }
    while (result > 1000) {
        result -= 100
    }
    try {
        result /= a
    } catch (ex: ArithmeticException) {
        result = 0
    }
    return if (d) result else result + name.length
}

fun deeplyNestedFunction(items: List<Int>): Int {
    var total = 0
    for (i in items) {
        if (i > 0) {
            for (j in 0 until i) {
                if (j % 2 == 0) {
                    while (total < 100) {
                        total += j
                    }
                }
            }
        }
    }
    return total
}
