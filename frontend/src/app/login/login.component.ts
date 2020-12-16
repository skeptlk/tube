import { Component } from '@angular/core';
import { FormControl, FormGroup, Validators } from '@angular/forms';
import { Router } from '@angular/router';
import { first } from 'rxjs/operators';
import { AuthService, NotificationService } from '../services';

@Component({
    selector: 'login',
    templateUrl: './login.component.html',
    styleUrls: ['./login.component.scss']
})
export class LoginComponent {

    loginForm = new FormGroup({
        name: new FormControl('', [Validators.required]),
        password: new FormControl('', [Validators.required])
    })

    constructor(
        private notifService: NotificationService,
        private auth: AuthService,
        private router: Router,
    ) { }

    submit() {
        if (this.loginForm.valid) {
            this.auth.login(this.loginForm.value)
                .pipe(first())
                .subscribe(
                    () => this.router.navigate(['']),
                    (error) => this.notifService.error(error));
        }
    }

}
