/*
 * Copyright (c) 2018
 * Mainflux
 *
 * SPDX-License-Identifier: Apache-2.0
 */
 
import { Component, OnInit } from '@angular/core';
import { FormBuilder, FormGroup, Validators } from '@angular/forms';

import { UiStore } from '../../../core/store/ui.store';
import { AuthStore } from '../../../core/store/auth.store';

@Component({
  selector: 'app-login',
  templateUrl: './login.component.html',
  styleUrls: ['./login.component.scss']
})
export class LoginComponent implements OnInit {
  loginForm: FormGroup;

  constructor(
    private fb: FormBuilder,
    private uiStore: UiStore,
    private authStore: AuthStore,
  ) { }

  ngOnInit() {
    this.loginForm = this.fb.group({
      email: ['', [Validators.required, Validators.email]],
      password: ['', [Validators.required]],
    });
  }

  login() {
    this.authStore.login(this.getUserDataFromForm());
  }

  signup() {
    this.uiStore.goToSignup();
  }

  getUserDataFromForm() {
    return {
      email: this.loginForm.get('email').value,
      password: this.loginForm.get('password').value
    };
  }
}
