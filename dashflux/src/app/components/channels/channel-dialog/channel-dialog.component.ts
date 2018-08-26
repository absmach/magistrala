/*
 * Copyright (c) 2018
 * Mainflux
 *
 * SPDX-License-Identifier: Apache-2.0
 */
 
import { Component, EventEmitter, Inject, OnInit, Output } from '@angular/core';
import { FormBuilder, FormGroup } from '@angular/forms';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material';
import { toJS } from 'mobx';

import { ThingsStore } from '../../../core/store/things.store';
import { Channel, Thing } from '../../../core/store/models';

@Component({
  selector: 'app-channel-dialog',
  templateUrl: './channel-dialog.component.html',
  styleUrls: ['./channel-dialog.component.scss']
})
export class ChannelDialogComponent implements OnInit {
  addChannelForm: FormGroup;
  @Output() submit: EventEmitter<Channel> = new EventEmitter<Channel>();
  editMode: boolean;

  constructor(
    private fb: FormBuilder,
    private dialogRef: MatDialogRef<ChannelDialogComponent>,
    @Inject(MAT_DIALOG_DATA) public data: Channel,
    public thingsStore: ThingsStore,
  ) { }

  ngOnInit() {
    this.addChannelForm = this.fb.group(
      {
        id: [''],
        name: [''],
        connected: [[]]
      }
    );

    if (this.data) {
      this.editMode = true;
      this.addChannelForm.patchValue(toJS(this.data));
    } else {
      this.editMode = false;
    }
  }

  onAddChannel() {
    const channel = this.addChannelForm.value;
    this.submit.emit(channel);
    this.dialogRef.close();
  }

  compareFunction(obj1: Thing, obj2: Thing) {
    return obj1.id === obj2.id;
  }
}
