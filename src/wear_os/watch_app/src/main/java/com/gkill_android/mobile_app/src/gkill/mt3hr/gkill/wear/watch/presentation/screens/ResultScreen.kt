package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.presentation.screens

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.wear.compose.material.Button
import androidx.wear.compose.material.Text
import com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.R
import kotlinx.coroutines.delay

/**
 * Shows the result of the KFTL submission.
 * - success=true  → green check, R.string.result_success
 * - success=false → error message
 */
@Composable
fun ResultScreen(
    success: Boolean,
    errorMessage: String = "",
    onDismiss: () -> Unit
) {
    if (success) {
        LaunchedEffect(Unit) {
            delay(2000L)
            onDismiss()
        }
    }
    Column(
        modifier = Modifier
            .fillMaxSize()
            .padding(16.dp),
        verticalArrangement = Arrangement.Center,
        horizontalAlignment = Alignment.CenterHorizontally
    ) {
        if (success) {
            Text(
                text = stringResource(R.string.result_success),
                textAlign = TextAlign.Center,
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(bottom = 12.dp)
            )
        } else {
            Text(
                text = stringResource(R.string.result_error, errorMessage),
                textAlign = TextAlign.Center,
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(bottom = 12.dp)
            )
        }
        Button(onClick = onDismiss) {
            Text(stringResource(R.string.back))
        }
    }
}
