package com.example.alpha

import com.example.beta.Beta
import com.thirdparty.http.Client
import java.util.UUID
import kotlin.math.abs
import kotlinx.coroutines.flow.Flow

class Alpha {
    fun id(): String = UUID.randomUUID().toString()

    fun magnitude(x: Int): Int = abs(x)
}
