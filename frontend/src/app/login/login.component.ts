import { Component, OnInit } from '@angular/core';
import { FormControl, FormGroup, Validators } from '@angular/forms';
import { Router } from '@angular/router';
import { first } from 'rxjs/operators';
import { AuthService, ErrorService } from '../services';

@Component({
    selector: 'app-login',
    templateUrl: './login.component.html',
    styleUrls: ['./login.component.scss']
})
export class LoginComponent implements OnInit {

    loginForm = new FormGroup({
        name: new FormControl('', [Validators.required]),
        password: new FormControl('', [Validators.required])
    })

    constructor(
        private errorService: ErrorService,
        private auth: AuthService,
        private router: Router,
    ) { }

    ngOnInit(): void {
    }

    submit() {
        if (this.loginForm.valid) {
            console.log(this.loginForm.value);

            this.auth.login(this.loginForm.value)
                .pipe(first())
                .subscribe(
                    () => {
                        this.router.navigate(['']);
                    },
                    error => this.errorService.error(error));
        }
    }

}
