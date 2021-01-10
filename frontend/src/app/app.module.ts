import { BrowserModule } from '@angular/platform-browser';
import { NgModule } from '@angular/core';
import { AppRoutingModule } from './app-routing.module';
import { ClarityModule } from '@clr/angular';
import { ReactiveFormsModule } from '@angular/forms';
import { FormsModule } from '@angular/forms';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';

import { HttpClientModule, HTTP_INTERCEPTORS } from '@angular/common/http';
import { JwtInterceptor, ErrorInterceptor } from './helpers';

import { DateAgoPipe } from 'src/pipes/date-ago.pipe';


import { AppComponent } from './app.component';
import { HomeComponent } from './home/home.component';
import { AdminComponent } from './admin/admin.component';
import { AccountComponent } from './account/account.component';
import { UploadComponent } from './upload/upload.component';
import { SearchComponent } from './search/search.component';
import { LoginComponent } from './login/login.component';
import { SignupComponent } from './signup/signup.component';
import { VideoComponent } from './video/video.component';
import { VideoOwnerControlsComponent } from './video-owner-controls/video-owner-controls.component';
import { CommentsComponent } from './comments/comments.component';
import { SingleCommentComponent } from './comments/single-comment/single-comment.component';
import { UserProfileComponent } from './user-profile/user-profile.component';
import { AdminUsersListComponent } from './admin/admin-users-list/admin-users-list.component';
import { AdminVideosListComponent } from './admin/admin-videos-list/admin-videos-list.component';
import { AdminUsersChartComponent } from './admin/admin-users-chart/admin-users-chart.component';
import { BestVideosComponent } from './best-videos/best-videos.component';
import { NgSelectModule } from '@ng-select/ng-select';
import { VideoEditComponent } from './video/video-edit/video-edit.component';


@NgModule({
    declarations: [
        DateAgoPipe,
        AppComponent,
        HomeComponent,
        AdminComponent,
        AccountComponent,
        UploadComponent,
        SearchComponent,
        LoginComponent,
        SignupComponent,
        VideoComponent,
        VideoOwnerControlsComponent,
        CommentsComponent,
        SingleCommentComponent,
        UserProfileComponent,
        AdminUsersListComponent,
        AdminVideosListComponent,
        AdminUsersChartComponent,
        BestVideosComponent,
        VideoEditComponent
    ],
    imports: [
        BrowserModule,
        AppRoutingModule,
        HttpClientModule,
        ClarityModule,
        BrowserAnimationsModule,
        ReactiveFormsModule,
        FormsModule,
        NgSelectModule
    ],
    providers: [
        { provide: HTTP_INTERCEPTORS, useClass: JwtInterceptor, multi: true },
        { provide: HTTP_INTERCEPTORS, useClass: ErrorInterceptor, multi: true },
    ],
    bootstrap: [AppComponent]
})
export class AppModule { };
