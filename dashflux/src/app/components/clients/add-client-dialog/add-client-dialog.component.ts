import { Component, EventEmitter, Inject, OnInit, Output } from '@angular/core';
import { AbstractControl, FormBuilder, FormGroup, Validators } from '@angular/forms';
import { MAT_DIALOG_DATA, MatDialogRef } from '@angular/material';

import { Client } from '../../../core/store/models';

@Component({
  selector: 'app-add-client-dialog',
  templateUrl: './add-client-dialog.component.html',
  styleUrls: ['./add-client-dialog.component.scss']
})
export class AddClientDialogComponent implements OnInit {
  addClientForm: FormGroup;
  @Output() submit: EventEmitter<Client> = new EventEmitter<Client>();

  constructor(
    private fb: FormBuilder,
    private dialogRef: MatDialogRef<AddClientDialogComponent>,
    @Inject(MAT_DIALOG_DATA) public data: Client
  ) { }

  ngOnInit() {
    this.addClientForm = this.fb.group(
      {
        id: [''],
        type: ['', [Validators.required]],
        name: ['', [Validators.required, Validators.minLength(5)]],
        payload: ['']
      }
    );

    if (this.data) {
      this.addClientForm.patchValue(this.data);
      this.addClientForm.get('payload').patchValue(JSON.stringify(this.data.payload));
    }
  }

  onAddClient() {
    const client = this.addClientForm.value;
    this.submit.emit(client);
    this.dialogRef.close();
  }
}
