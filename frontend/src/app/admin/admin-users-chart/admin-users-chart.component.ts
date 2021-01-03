import { Component, OnInit } from '@angular/core';
import { User, Video, UserStats } from 'src/app/models';
import { AdminService } from 'src/app/services';

@Component({
    selector: 'admin-users-chart',
    templateUrl: './admin-users-chart.component.html'
})
export class AdminUsersChartComponent implements OnInit {

    users: UserStats[] = [];

    constructor(private adminService: AdminService) { }

    async ngOnInit() {
        this.users = await this.adminService.getUsersChart().toPromise();
    }



}
