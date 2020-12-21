import { Component, OnInit } from '@angular/core';
import { AuthService } from './services';

@Component({
  selector: 'app-root',
  templateUrl: './app.component.html',
  styleUrls: ['./app.component.scss']
})
export class AppComponent implements OnInit {

    constructor(public auth: AuthService) {

    }

    ngOnInit() {
        console.log(this.auth.currentUserValue);
        
    }

}
