import { Component, EventEmitter, Inject, OnInit, Output } from '@angular/core';
import { FormBuilder, FormGroup } from '@angular/forms';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material';
import { toJS } from 'mobx';

import { ClientsStore } from '../../../core/store/clients.store';
import { Channel, Client } from '../../../core/store/models';

@Component({
  selector: 'app-add-channel-dialog',
  templateUrl: './add-channel-dialog.component.html',
  styleUrls: ['./add-channel-dialog.component.scss']
})
export class AddChannelDialogComponent implements OnInit {
  addChannelForm: FormGroup;
  @Output() submit: EventEmitter<Channel> = new EventEmitter<Channel>();
  editMode: boolean;

  constructor(
    private fb: FormBuilder,
    private dialogRef: MatDialogRef<AddChannelDialogComponent>,
    @Inject(MAT_DIALOG_DATA) public data: Channel,
    public clientsStore: ClientsStore,
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

  compareFunction(obj1: Client, obj2: Client) {
    return obj1.id === obj2.id;
  }
}
