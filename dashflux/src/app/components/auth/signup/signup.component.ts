/*
 * Copyright (c) 2018
 * Mainflux
 *
 * SPDX-License-Identifier: Apache-2.0
 */
 
import { Component, OnInit } from '@angular/core';
import { AbstractControl, FormBuilder, FormGroup, Validators } from '@angular/forms';

import { AuthStore } from '../../../core/store/auth.store';
import { UiStore } from '../../../core/store/ui.store';

@Component({
  selector: 'app-signup',
  templateUrl: './signup.component.html',
  styleUrls: ['./signup.component.scss']
})
export class SignupComponent implements OnInit {
  signupForm: FormGroup;

  constructor(
    private fb: FormBuilder,
    private uiStore: UiStore,
    private authStore: AuthStore,
  ) { }

  ngOnInit() {
    this.signupForm = this.fb.group({
      email: ['', [Validators.required, Validators.email]],
      passwords: this.fb.group({
        password: ['', [Validators.required]],
        repeatPassword: ['', [Validators.required]]
      }, { validator: this.comparePasswords })
    });
  }

  comparePasswords(c: AbstractControl): { [key: string]: boolean } {
    const pass = c.get('password');
    const repeatPassword = c.get('repeatPassword');

    if (pass.value !== repeatPassword.value) {
      return { 'match': true };
    }
    return null;
  }

  signup() {
    this.authStore.signup(this.getUserDataFromForm());
  }

  login() {
    this.uiStore.goToLogin();
  }

  getUserDataFromForm() {
    return {
      email: this.signupForm.get('email').value,
      password: this.signupForm.get('passwords.password').value
    };
  }
}
